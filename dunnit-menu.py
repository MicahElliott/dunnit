import rumps
import os
import os.path

rumps.debug_mode(True)

nightyfile = "/tmp/dunnit-nighty"
if os.path.isfile(nightyfile): os.remove(nightyfile)

class DunnitStatusBarApp(rumps.App):

    # @rumps.clicked("Preferences (ignore!)")
    # def prefs(self, _):
    #     rumps.alert("jk! no preferences available!")

    @rumps.clicked("TODO")
    def todo(self, _):
        os.system("~/dunnit/dunnit-todo")
        # Use todoist instead
        # os.system('cliclick kd:cmd,ctrl t:a ku:cmd,ctrl')


    @rumps.clicked("Dunnit")
    def bubble(self, sender):
        if not sender.state:
            os.system("~/dunnit/dunnit-bubble")
            # Use todoist instead
            # os.system('cliclick kd:cmd,ctrl t:a ku:cmd,ctrl')

    @rumps.clicked("Show today's progress")
    def progress(self, _):
        prog = os.popen("~/dunnit/dunnit-progress").read()
        win = rumps.Window("foo", 'bar', dimensions=(500,600))
        win.title = 'Dunnit Progress Today'
        win.message = "This is an in-flight view of your day so far. It's just markdown; you'll have a chance to edit it when you close the day."
        win.default_text = prog
        resp = win.run()
        print(resp)
        # Use todoist instead
        # os.system('cliclick kd:cmd,ctrl t:t ku:cmd,ctrl')

    @rumps.clicked("Edit/summarize the day")
    def eod(self, _):
        os.system("~/dunnit/dunnit-eod")
        # Use todoist instead
        # os.system('cliclick kd:cmd,ctrl t:t ku:cmd,ctrl')

    @rumps.clicked("Nighty-night mode")
    def onoff(self, sender):
        if sender.state: # night mode is on
            print("Turning off nighty mode")
            sender.state = False
            self.title = '✔ Dunnit'
            if os.path.isfile(nightyfile): os.remove(nightyfile)
            os.system("~/dunnit/dunnit-bod")
        else:
            print("Turning on nighty mode")
            self.title = '💤 Dunnit'
            with open(nightyfile, 'a'): pass
            sender.state = True

    @rumps.clicked("Generate daily report")
    def report(self, _):
        os.system("~/dunnit/dunnit-report")
        # Use todoist instead
        # os.system('cliclick kd:cmd,ctrl t:t ku:cmd,ctrl')

    # TODO
    @rumps.clicked("Email daily report")
    def email(self, _):
        os.system("~/dunnit/dunnit-email")

    @rumps.clicked("About Dunnit")
    def about(self, _):
        rumps.notification("Dunnit is for tracking WTF you did", "Just write a sentence each hour.", "You can pop up and record anytime.")

    # @rumps.clicked('On')
    # def button(self, sender):
    #     sender.title = 'Off' if sender.title == 'On' else 'On'
    #     Window("I can't think of a good example app...").run()


if __name__ == "__main__":
    DunnitStatusBarApp("✔ Dunnit").run()
