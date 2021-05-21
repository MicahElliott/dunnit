import rumps
import os
import os.path
import datetime

rumps.debug_mode(True)

nightyfile = "/tmp/dunnit-nighty"
if os.path.isfile(nightyfile): os.remove(nightyfile)

class DunnitStatusBarApp(rumps.App):

    # @rumps.clicked("✅🚀 Dunnit (YAY!)")
    @rumps.clicked("🚀 Dunnit (YAY!)")
    def bubble(self, sender):
        # if not sender.state:
        os.system("~/dunnit/dunnit-bubble frommenu")

    @rumps.clicked("🚧 Blocker")
    def blocker(self, sender):
        # if not sender.state:
        os.system("~/dunnit/dunnit-blocker")

    @rumps.clicked("🗓 Planning", "💡 Todo")
    def todo(self, _):
        os.system("~/dunnit/dunnit-todo")
        # Use todoist instead
        # os.system('cliclick kd:cmd,ctrl t:a ku:cmd,ctrl')

    @rumps.clicked("🗓 Planning", "🍅 Timer")
    def pomodoro(self, _):
        es = os.system("~/dunnit/dunnit-pomodoro")

    @rumps.clicked("🗓 Planning", "🥅 Goals")
    def setgoals(self, _):
        os.system("~/dunnit/dunnit-goals")

    @rumps.clicked("🗓 Planning", "🎯 Weekly Objectives")
    def objectives(self, _):
        os.system("~/dunnit/dunnit-objectives")

    @rumps.clicked("🗓 Planning", "👀 All")
    def showtodos(self, _):
        with open('help.txt', 'r') as file: txt = file.read()
        prog = os.popen("~/dunnit/dunnit-showtodos").read()
        win = rumps.Window("foo", 'bar', dimensions=(500,600))
        win.title = 'Dunnit Todos'
        win.message = "These are all the things you planned to do."
        win.default_text = prog
        resp = win.run()

    @rumps.clicked("🧍‍♀️ Standup")
    def standup(self, _):
        os.system("~/dunnit/dunnit-standup")

    @rumps.clicked("📒 Ledger", "👀 View")
    def progress(self, _):
        prog = os.popen("~/dunnit/dunnit-progress").read()
        win = rumps.Window("foo", 'bar', dimensions=(500,600))
        win.title = 'Dunnit Progress Today'
        win.message = "This is an in-flight view of your day so far. It's just a ledger of raw entries; you'll have a chance to edit it in a better format when you close the day."
        win.default_text = prog
        resp = win.run()
        print(resp)

    @rumps.clicked("📒 Ledger", "✏️ Edit (careful!)")
    def raw(self, _):
        es = os.system("~/dunnit/dunnit-editraw")

    @rumps.clicked("🌯 Wrap Up", "🏁 Finalize and Edit Today")
    def eod(self, _):
        es = os.system("~/dunnit/dunnit-eod")
        if es != 0: rumps.alert('Summary file already exists! Delete it and try again.')

    @rumps.clicked("🌯 Wrap Up", "📓 Daily Report (html)")
    def report(self, _):
        os.system("~/dunnit/dunnit-report")

    @rumps.clicked("🌯 Wrap Up", "📧 Daily Report (email)")
    def email(self, _):
        os.system("~/dunnit/dunnit-email")

    # TODO
    @rumps.clicked("🌯 Wrap Up", "🎉 Weekly Report")
    def weeklyreport(self, _):
        today = datetime.datetime.today()
        ww = today.strftime("%U")
        # ww = datetime.date(2010, 6, 16).isocalendar()[1]
        os.system(f"~/dunnit/dunnit-eowsummary w{ww}")

    @rumps.clicked("☁️ Sync", "⬆ Push")
    def push(self, _):
        rumps.notification("Pushing to remote", "...", "...")
        os.system("~/dunnit/dunnit-push")

    @rumps.clicked("☁️ Sync", "⬇ Pull")
    def pull(self, _):
        rumps.notification("Pulling from remote", "...", "...")
        os.system("~/dunnit/dunnit-pull")

    @rumps.clicked("⚙️ Misc", "🛠 Preferences")
    def preferences(self, _):
        os.system("~/dunnit/dunnit-preferences")

    @rumps.clicked("⚙️ Misc", "📖 Full Tutorial")
    def help(self, _):
        with open('help.txt', 'r') as file: txt = file.read()
        win = rumps.Window("foo", 'bar', dimensions=(500,600))
        win.title = 'Dunnit Help'
        win.message = "All about Dunnit and its usage'"
        win.default_text = txt
        resp = win.run()

    @rumps.clicked("⚙️ Misc", "ℹ️ Version")
    def verinfo(self, _):
        rumps.notification("Version", "0.0.1", "Something")

    @rumps.clicked("⚙️ Misc", "✨ Check for Update")
    def swupdate(self, _):
        os.system("~/dunnit/dunnit-swupdate")

    @rumps.clicked("⚙️ Misc", "❓ What's Dunnit?")
    def about(self, _):
        rumps.notification("Dunnit is for tracking WTF you did", "Just write a sentence each hour.", "You can pop up and record anytime.")

    @rumps.clicked("💤 AFK (disable)")
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

    # @rumps.clicked('On')
    # def button(self, sender):
    #     sender.title = 'Off' if sender.title == 'On' else 'On'
    #     Window("I can't think of a good example app...").run()


if __name__ == "__main__":
    DunnitStatusBarApp("✔ Dunnit").run()
