Output demo/demo.gif

Set Shell "bash"
Set FontSize 14
Set Width 900
Set Height 500
Set Padding 20
Set Theme "Catppuccin Mocha"

# Title
Type "# chrest: manage your browser from the terminal"
Enter
Sleep 1s

# Initialize
Type "chrest init --browser chrome"
Enter
Sleep 2s

# List windows
Type "chrest list-windows | jq '.[0] | {id, state, tabs: [.tabs[] | {title, url}][:3]}'"
Enter
Sleep 3s

# Create a tab
Type "chrest create-tab --url https://github.com"
Enter
Sleep 2s

# List tabs (show count)
Type "chrest list-tabs | jq '.[0] | length'"
Enter
Sleep 2s

# Close the tab we created
Type "chrest close-tab --tab_id $(chrest list-tabs | jq '.[0][-1].id')"
Enter
Sleep 2s

# Get browser info
Type "chrest browser-info | jq '.[0].browser'"
Enter
Sleep 2s

Sleep 1s
