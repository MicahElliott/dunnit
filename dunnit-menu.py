import rumps
import os
import os.path

nightyfile = "/tmp/dunnit-nighty"
if os.path.isfile(nightyfile): os.remove(nightyfile)

class DunnitStatusBarApp(rumps.App):

    @rumps.clicked("About Dunnit")
    def about(self, _):
        rumps.notification("Dunnit is for tracking WTF you did", "Just write a sentence each hour.", "You can pop up and record anytime.")

    # @rumps.clicked("Preferences (ignore!)")
    # def prefs(self, _):
    #     rumps.alert("jk! no preferences available!")

    @rumps.clicked("Nighty-night mode")
    def onoff(self, sender):
        if sender.state: # night mode is on
            print("Turning off nighty mode")
            sender.state = False
            if os.path.isfile(nightyfile): os.remove(nightyfile)
        else:
            print("Turning on nighty mode")
            with open(nightyfile, 'a'): pass
            sender.state = True

    @rumps.clicked("Add to TODO target")
    def todo(self, _):
        os.system("~/dunnit/dunnit-todo")

    @rumps.clicked("Record a dunnit")
    def bubble(self, sender):
        if not sender.state:
            os.system("~/dunnit/dunnit-bubble")

    @rumps.clicked("Edit/summarize the day")
    def eod(self, _):
        os.system("~/dunnit/dunnit-eod")

if __name__ == "__main__":
    DunnitStatusBarApp("Dunnit").run()
