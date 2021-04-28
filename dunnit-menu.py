import rumps
import os
import os.path

rumps.debug_mode(True)

nightyfile = "/tmp/dunnit-nighty"
if os.path.isfile(nightyfile): os.remove(nightyfile)

class DunnitStatusBarApp(rumps.App):

    @rumps.clicked("GM, Sunshine! (set goals)")
    def setgoals(self, _):
        os.system("~/dunnit/dunnit-goals")
        # Use todoist instead
        # os.system('cliclick kd:cmd,ctrl t:t ku:cmd,ctrl')

    @rumps.clicked("All Todos")
    def showtodos(self, _):
        with open('help.txt', 'r') as file: txt = file.read()
        prog = os.popen("~/dunnit/dunnit-showtodos").read()
        win = rumps.Window("foo", 'bar', dimensions=(500,600))
        win.title = 'Dunnit Todos'
        win.message = "These are all the things you planned to do."
        win.default_text = prog
        resp = win.run()

    @rumps.clicked("Todo")
    def todo(self, _):
        os.system("~/dunnit/dunnit-todo")
        # Use todoist instead
        # os.system('cliclick kd:cmd,ctrl t:a ku:cmd,ctrl')

    @rumps.clicked("Dunnit (YAY!)")
    def bubble(self, sender):
        # if not sender.state:
        os.system("~/dunnit/dunnit-bubble")
        # Use todoist instead
        # os.system('cliclick kd:cmd,ctrl t:a ku:cmd,ctrl')

    @rumps.clicked("Today's Ledger")
    def progress(self, _):
        prog = os.popen("~/dunnit/dunnit-progress").read()
        win = rumps.Window("foo", 'bar', dimensions=(500,600))
        win.title = 'Dunnit Progress Today'
        win.message = "This is an in-flight view of your day so far. It's just a ledger of raw entries; you'll have a chance to edit it in a better format when you close the day."
        win.default_text = prog
        resp = win.run()
        print(resp)
        # Use todoist instead
        # os.system('cliclick kd:cmd,ctrl t:t ku:cmd,ctrl')

    @rumps.clicked("Edit Ledger Items (careful!)")
    def raw(self, _):
        es = os.system("~/dunnit/dunnit-editraw")

    @rumps.clicked("Thx for all the fish! (finalize)")
    def eod(self, _):
        # TODO pop lp alert if already exists
        es = os.system("~/dunnit/dunnit-eod")
        if es != 0: rumps.alert('Summary file already exists! Delete it and try again.')
        # Use todoist instead
        # os.system('cliclick kd:cmd,ctrl t:t ku:cmd,ctrl')

    @rumps.clicked("Daily Report (html)")
    def report(self, _):
        os.system("~/dunnit/dunnit-report")
        # Use todoist instead
        # os.system('cliclick kd:cmd,ctrl t:t ku:cmd,ctrl')

    # TODO
    @rumps.clicked("Daily Report (email, NYI)")
    def email(self, _):
        os.system("~/dunnit/dunnit-email")

    @rumps.clicked("Nighty-night Mode (AFK)")
    def onoff(self, sender):
        if sender.state: # night mode is on
            print("Turning off nighty mode")
            sender.state = False
            self.title = '✔ Dunnit'
            if os.path.isfile(nightyfile): os.remove(nightyfile)
            # os.system("~/dunnit/dunnit-goals")
        else:
            print("Turning on nighty mode")
            self.title = '💤 Dunnit'
            with open(nightyfile, 'a'): pass
            sender.state = True

    # @rumps.clicked("Preferences (NYI)")
    # def prefs(self, _):
    #     rumps.alert("jk! No preferences available yet.")

    @rumps.clicked("Help")
    def help(self, _):
        with open('help.txt', 'r') as file: txt = file.read()
        win = rumps.Window("foo", 'bar', dimensions=(500,600))
        win.title = 'Dunnit Help'
        win.message = "All about Dunnit and its usage'"
        win.default_text = txt
        resp = win.run()


    @rumps.clicked("About")
    def about(self, _):
        rumps.notification("Dunnit is for tracking WTF you did", "Just write a sentence each hour.", "You can pop up and record anytime.")

    # @rumps.clicked('On')
    # def button(self, sender):
    #     sender.title = 'Off' if sender.title == 'On' else 'On'
    #     Window("I can't think of a good example app...").run()


if __name__ == "__main__":
    DunnitStatusBarApp("✔ Dunnit").run()
