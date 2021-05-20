#! /bin/zsh

mkdir -p /tmp/dunnit
cd /tmp/dunnit

print "Installing brew packages"
brew install direnv bat fzf coreutils git grep python wget gsed terminal-notifier the_silver_searcher pandoc

print "Installing alerter for pop-up alerts"
wget https://github.com/vjeantet/alerter/releases/download/004/alerter_v004_darwin_amd64.zip
unzip alerter_v004_darwin_amd64.zip
# FIXME mac might not have /usr/local yet??
mv alerter /usr/local/bin/alerter
alerter -message 'Congrats if you can see this!'

print "Installing python packages"
pip3 install rumps

print "Running Dunnit for the first time"
cd ~/dunnit
./dunnit-bubble

print 'Activating default settings'
gcp --no-clobber config-templates/* .

print 'Set up your preferences'
source dunnit.zsh
dunnit-preferences

# print "Loading background processes for hourly notifications"
# launchctl load -w ~/Library/LaunchAgents/dunnit*.plist

print "Starting the Dunnit menu"
./dunnit-menu

print "You can close this terminal now"
