#! /bin/zsh

mkdir -p /tmp/dunnit
cd /tmp/dunnit

print "Installing alerter for pop-up alerts"
wget https://github.com/vjeantet/alerter/releases/download/004/alerter_v004_darwin_amd64.zip
unzip alerter_v004_darwin_amd64.zip
mv alerter /usr/local/bin/alerter
alerter -message 'Congrats if you can see this!'

print "Installing brew packages"
brew install direnv bat fzf coreutils git grep python wget gsed terminal-notifier the_silver_searcher pandoc

print "Installing python packages"
pip3 install rumps

print "Running Dunnit for the first time"
cd ~/dunnit
./dunnit-bubble

print "Loading background processes for hourly notifications"
launchctl load -w dunnit.plist
launchctl load -w dunnit-eod.plist

print "Starting the Dunnit menu"
./dunnit-menu

print "You can close this terminal now"
