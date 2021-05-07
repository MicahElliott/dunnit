# Configurable Settings for Dunnit
# Just plain zsh environment variables
# NOTE: Dunnit mostly only runs on weekdays

# Approximately when your day starts and ends
DUNNIT_DAY_START=(08 00)
DUNNIT_DAY_END=(17 05)

# The minute of every hour that popups present
DUNNIT_HOURLY=58

# When you usually take lunch, goal reminder alert will present
DUNNIT_LUNCHTIME=(11 30)

# The base path to where you keep your sync'd trackables (ledgers and summaries)
DUNNIT_MYDUNNITS_REPO="https://github.com/$USER/mydunnits"

# When your stand-up meeting happens each day, yesterday's report pops up then
DUNNIT_STANDUP=(10 15)

# If you use Jira, this base path finds and links your stories
DUNNIT_JIRA_ROOT='https://jira.mycompany.com'
