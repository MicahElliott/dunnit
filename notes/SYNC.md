Usage model:

- each day gets a dated file that can live/grow on multiple machines
- sync across 2 machines
- log-style file
- only a couple times per day
- never concurrent
- only additive entries, but an entry could be edited on the machine in which it originated
- ok to always pull before push
- one-line entries only, timestamped to the minute
- all entries are obviously unique (no similar lines)
- would be nice to avoid pulling in a git package

Not sure where to sync to; could be:

- shared mounted dir (dropbox, icloud, etc)
- or to service (to local file that's push to service (dropbox, etc))
- git
- does it matter?

I wrote to LLM:

> I don't care about order on entries in file. Also, no similar entry would
> show up from the two sources, so no almost-matching lines problems. I'd
> rather not have to read before every write, but maybe that's fine too --
> it's more about push/pull syncing. So I'd guess it doesn't matter with this
> model if it's dropbox or icloud or local-with-manual-push-pull, right?

Its response:

> Correct. If you do not care about the order of lines and the lines are
> entirely unique, the underlying storage layer becomes irrelevant. Whether it
> is Dropbox, iCloud, a Git repository, or a manual file-transfer script, the
> core synchronization rules remain exactly the same.

> Because you want to avoid reading the remote file on every single local log
> write (which slows down performance), you can decouple your app into a
> Local-First Write system with a Periodic Sync Step.

Let me know your thinking on how to simply design this.

---

From a gemini session (discussing using dropbox/icloud as backing store on mobile/everywhere):

To clarify, yes, you still have to deal with tricky iOS URL mechanics, but because you are using Fyne, Fyne wraps all the platform-specific iOS UI and code for you. You do not have to write a single line of Objective-C, Swift, or custom cgo code. [1] 
Here is exactly how Go handles this and how the user permission architecture reconciles with Fyne's cross-platform storage rules: [2] 

## 1. How does Go "create a single piece of iOS UI"?
When you use a cross-platform framework like Fyne, it links to native iOS system frameworks under the hood. You invoke standard Go functions, and Fyne maps them automatically: [2] 

* On Desktop (Linux/macOS): Fyne pops up a standard desktop file/folder picker window.
* On iOS: Fyne translates that exact same Go call into the native iOS UIDocumentPickerViewController (the system Files app modal). [3, 4] 

You do not write "iOS-specific UI." You write pure Fyne Go code, and iOS
handles the visual prompt to grant access to iCloud or Dropbox. [3, 5]

## 2. The Fyne Solution (Code Example)
Instead of standard string file paths (like /user/documents), Fyne forces you to pass around a fyne.URI object. This abstraction is exactly how Fyne hides the "tricky iOS security URLs" from your standard logic. [1, 2] 
When a user triggers a first-time setup in your app, you use Fyne's dialog.NewFolderOpen: [6] 

```
package main
import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
)
func main() {
	myApp := app.NewWithID("com.yourcompany.mytreeapp")
	myWindow := myApp.NewWindow("Tree Sync Setup")

	// 1. Create a button to let the user select their sync folder
	// On iOS, this opens the system Document Picker into iCloud/Dropbox
	dialog.ShowFolderOpen(func(dir fyne.ListableURI, err error) {
		if err != nil || dir == nil {
			return // User canceled
		}

		// 2. Fyne automatically handles the iOS security-scoped authorization block.
		// You save the URI as a string string in Fyne's built-in cross-platform preferences.
		myApp.Preferences().SetString("sync_folder_uri", dir.String())
		
		fmt.Println("Successfully bookmarked directory:", dir.String())
	}, myWindow)

	myWindow.ShowAndRun()
}
```

## 3. Reading and Writing Daily Files (The Persistent Bookmark)
On subsequent launches, your app reads the string out of Fyne's preferences. When Fyne decodes that URI on iOS, it automatically hooks back into Apple's persistent Security-Scoped Bookmarks. [2, 3, 7] 
You can use Fyne's storage package to interact with the tree inside that folder recursively: [8] 

```
// Retrieve the root folder from preferences
savedURIStr := myApp.Preferences().String("sync_folder_uri")
rootURI, _ := storage.ParseURI(savedURIStr)
rootFolder, _ := storage.ListerForURI(rootURI)
// Create or access a daily file (e.g., 2026-09-08.json)
dailyFileURI, _ := storage.ChildURI(rootFolder.ToURI(), "2026-09-08.json")
// Open it for writing using Fyne's stream wrappers (handles iOS permissions natively)
writer, err := storage.Writer(dailyFileURI)if err == nil {
    writer.Write([]byte(`{"node": "root", "data": "today's tree text"}`))
    writer.Close()
}
```

## Reconciling the Logic

   1. First-Time User Interaction: The user hits a "Select Sync Folder" button in your Fyne UI. Fyne hands the flow to iOS. The user picks their synced folder (inside iCloud Drive, Dropbox, or a Syncthing local folder). [3, 4, 5] 
   2. Behind the Scenes: iOS grants the app sandboxed permission. Fyne translates this into a standard Go fyne.URI string. You save that string. [2, 7, 9] 
   3. Everyday Use: Your Go app relies solely on Fyne's storage package. As long as you use Fyne's stream readers/writers instead of standard Go os.OpenFile(), Fyne keeps the iOS security access active in the background. [1, 2, 8, 10] 

This approach allows you to achieve your goal: keeping your business logic purely in cross-platform Go, while Fyne quietly navigates the iOS security-scoped URL landscape for you. [1, 2] 
Would you like help setting up the Fyne App ID or iOS Entitlements configuration in your project file so the app is allowed to use external Document Pickers upon compilation? [11] 

[1] [https://github.com](https://github.com/fyne-io/fyne/discussions/3601)
[2] [https://github.com](https://github.com/fyne-io/fyne/wiki/Settings-API)
[3] [https://stackoverflow.com](https://stackoverflow.com/questions/70750276/how-to-enable-files-and-folders-permission-on-ios)
[4] [https://dev.to](https://dev.to/cjr29/learning-go-building-a-file-picker-using-fyneio-33le)
[5] [https://www.youtube.com](https://www.youtube.com/watch?v=iHH1fxBya-0)
[6] [https://docs.fyne.io](https://docs.fyne.io/api/v2/dialog/filedialog/)
[7] [https://fynelabs.com](https://fynelabs.com/2021/04/20/keep-all-your-desktop-fyne-apps-in-sync/)
[8] [https://github.com](https://github.com/fyne-io/fyne/issues/2546)
[9] [https://stackoverflow.com](https://stackoverflow.com/questions/75465021/permission-denied-when-trying-to-access-file-outside-of-ios-app-container)
[10] [https://github.com](https://github.com/fyne-io/fyne/discussions/3601)
[11] [https://docs.fyne.io](https://docs.fyne.io/started/mobile/)

---

> Is there anything that needs to be coded different depending on whether dropbox vs icloud? Hoping my app doesn't need to know anything about which syncing app is running.

No, your Go app does not need to know or care whether the user is using iCloud, Dropbox, or Syncthing. [1, 2]
Once the user selects the folder via the Fyne dialog, iOS abstracts the cloud provider away entirely. To your Go code, the selected folder looks and acts exactly the same regardless of the backend sync service.
However, to ensure a completely seamless experience without adding provider-specific logic, keep these two cross-platform rules in mind:

## 1. Avoid Relying on Instant "File-Watchers"
If you use file-system notification packages (like fsnotify in Go) to detect when a file has synced from another device, the behavior will vary wildly:

* iCloud notifies the OS instantly when a file changes.
* Dropbox/Syncthing might download the file in a temporary state first, triggering multiple erratic file events.

The Fix: Instead of listening for active filesystem changes, simply re-read the file when the user interacts with the app UI (e.g., when the app comes to the foreground or when a user clicks a refresh button).

## 2. Guard Against the "Missing File" (Cloud Eviction)
Both iCloud and Dropbox will occasionally "evict" old daily files to save disk space if the user hasn't opened them recently.

* To your Go code, an evicted file might temporarily look like it doesn't exist, or Fyne's storage.Reader might return a standard "file not available" error.

The Fix: You don't need provider-specific code to handle this. Just write a generic Go error handler for your file reads:

```
reader, err := storage.Reader(dailyFileURI)if err != nil {
    // Show a universal message to the user in Fyne
    dialog.ShowError(errors.New("File is archiving in the cloud. Please open the iOS Files app to wake it up."), myWindow)
    return
}
```

By letting iOS and Fyne handle the folder bridge, your Go code remains beautifully platform-agnostic.
If you'd like to test this setup, I can show you how to configure the Info.plist properties for your Fyne app compilation so iOS allows the app to open external cloud documents properly. Would that be helpful?

To ensure iOS allows your compiled Fyne app to access external cloud folders (like iCloud or Dropbox) and prevents files from becoming unreadable, you must configure specific keys in your iOS metadata bundle.
When compiling a Fyne application for iOS using the fyne package tool, you can provide a custom Info.plist file.
## 1. The Essential Info.plist Configuration
Create a file named Info.plist in your project root (or ensure these keys are added if you already have one). These settings tell iOS that your app intends to open and manage external documents:

```
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://apple.com. Landon">
<plist version="1.0">
<dict>
    <!-- 1. Allows the Fyne folder picker to access folders outside your app sandbox -->
    <key>UISupportsDocumentBrowser</key>
    <true/>

    <!-- 2. Tells iOS that this app can open documents in place rather than making isolated copies -->
    <key>LSSupportsOpeningDocumentsInPlace</key>
    <true/>

    <!-- 3. Standard Fyne configuration keys -->
    <key>CFBundleIdentifier</key>
    <string>com.yourcompany.mytreeapp</string>
    <key>CFBundleName</key>
    <string>MyTreeApp</string>
    <key>CFBundleShortVersionString</key>
    <string>1.0</string>
    <key>CFBundleVersion</key>
    <string>1</string>
</dict>
</plist>
```

## 2. How to Compile the App with Fyne
When you are ready to package your application for iOS testing or deployment, pass the -plist flag to the Fyne CLI tool so it bakes these rules into your app binary:

fyne package -os ios -appID com.yourcompany.mytreeapp -plist Info.plist

## 3. What to Expect During testing
When you launch this app on an iPhone or an iOS simulator and call dialog.ShowFolderOpen:

   1. A native modal will slide up showing locations like iCloud Drive, Dropbox, and On My iPhone.
   2. Once the user selects their target tree folder, iOS silently passes the authenticated security token to Fyne.
   3. Fyne handles the persistent bookmark generation automatically behind the scenes when you save the URI string.

Would you like help writing a small Go function to verify if a daily file exists inside that folder before trying to read it using Fyne's storage API?

