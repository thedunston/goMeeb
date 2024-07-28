# goMeeb

*Status: Actively Supported.*

goMeeb's purpose is to help with identifying anomalies across a set of computer systems.

goMeeb uses basic math functions (averages and logarithms) to help with identifying the anomalies.

## Use Cases

An organization has multiple systems (dozens, hundreds, or thousands) and want to determine if there are any running processes, loaded DLLS, local user accounts, scheduled tasks or just about any data point they want to examine to help find latent malicious artifacts.

Another use case is for an incident handler who saves data in CSV format and need to analyze the results and find anomalous data in a large dataset.

## File type

goMeeb works on CSV files.

## An Anomaly

What constitutes an anomaly is highly dependent on the environment and the data set collected. For example, if you collect running processes on 20 systems and have the 20 CSV files in a directory, the tool **meeb** will aggregate the list of all processes and print each one with the average number of times the process appears across 20 systems. If the process exists on all systems it will have an average of 1.0 next to it. If it appears on only one system, it will have 0.01 next to it. That means it appears on only 1% of those 20 systems. The process with 0.01 may not be an anomaly because it could be the one computer that has a different program running on it. Accordingly, it requires having a goal in mind and understanding the environment where the tools are used.

## Obtaining data in CSV format

CSV data can be obtained in a variety of ways. It can be exported from an existing databases, EDR tools, cli tools, or any place where you can get obtain data in a CSV format. You will also need at least 2 computers to compare the results. 

Powershell can export data to CSV using `Export-CSV -NoTypeInformation`. Use `-NoTypeInformation` to remove the extraneous text added to CSV files. Some Windows commands can export to CSV such as **tasklist** (`tasklist /v /fo csv > %COMPUTERNAME%-processes.csv`).

### Recommendations

Use tools that can export the full path of any process, driver, or DLL. Even when listing network sockets, obtain the full path of the process that has the socket open. It will provide much more valuable information and allows quickly detecting anomalies.

Try to include the hostname in the output because `mel` will list the entire CSV row and having the hostname in the output saves having to perform searches to see where the process exists.

When possible, collect data with the highest level of privileges available. Non-admin users may not have permissions to list the full path of some files. Setup an account that is used only for data collection and use that account when needed to elevate to an admin user. Monitor that account for usage since it should only be in use to run specific tools. Also, give it a name using the same scheme as regular users.

**Note that when using native cli tools like Powershell cmdlets, commands like tasklist, or wmic, some EDRs may trigger since those commands are also used by threat actors to enumerate a host.**

## Preparing to use goMeeb tools

It is best to run the tools on computers that are configured the same. For example, if you have 5 IIS webservers running on Windows 2019, then run the goMeeb tools on those systems. You can run them on an Exchange server and analyze with the WIndows 2019 servers, but it will have a lot of results you'll have to filter through because an IIS server and Exchange server have a different set of processes running and loaded modules. That may create a lot of work trying to verify if the data you're looking at is normal or an anomaly.

Also, run the tools on CSV files with the same type of data. If you have 20 computers and dump a list of running processes, then have a separate CSV files of running processes for each host using the same tool.

If you only have one computer running an IIS server on Windows 2016, a recommendation is to setup a stock version of Windows 2016 with the same version of IIS and any add-on programs. Then, use hte goMeeb's baseline tool to compare the output of the CSV files. For this example, it is best to use the **baseline** tool.

## Baseline tool Example

On a Windows 10 computer using Powershell, run:

`Get-Process | Select ProcessName,id,Path | Export-CSV -NoTypeInformation -Path "$env.ComputerName-processes.csv"`

which will produce output something like this:

`
"ProcessName","ID","Path"
"chrome.exe",1234,"C:\Users\user\AppData\Local\Google\Chrome\Application\chrome.exe"
"explorer.exe",5678,"C:\Windows\explorer.exe"
"systemd.exe",9012,"C:\Windows\System32\svchost.exe"
"spotify.exe",3456,"C:\Program Files\Spotify\Spotify.exe"
"Dwm.exe",2341,"C:\ProgramData\Dwm.exe"
"taskmgr.exe",7890,"C:\Windows\system32\taskeng.exe"
"wininit.exe",1111,"C:\Windows\system32\wininit.exe"
"cmd.exe",2222,"C:\Windows\System32\cmd.exe"
"chrome.exe",3333,"C:\Users\user\AppData\Local\Google\Chrome\Application\chrome.exe"
"dellupdate.exe",4444,"C:\Program Files (x86)\Dell Update\Update.exe"
"systemd.exe",5555,"C:\Windows\System32\svchost.exe"
`

On a Windows 10 system that is 'clean' or known to not be infected (here again, you can use a fresh install of Windows or the image used to configure new systems), run the same Powershell command and save it as the **baseline.csv** file.

`Get-Process | Select ProcessName,id,Path | Export-CSV -NoTypeInformation -Path "Windows10-baseline-processes.csv"`

and get output something like this.

"ProcessName","ID","Path"
"chrome.exe",2342,"C:\Users\user\AppData\Local\Google\Chrome\Application\chrome.exe"
"explorer.exe",4527,"C:\Windows\explorer.exe"
"systemd.exe",9819,"C:\Windows\System32\svchost.exe"
"spotify.exe",2543,"C:\Program Files\Spotify\Spotify.exe"
"taskmgr.exe",6532,"C:\Windows\system32\taskeng.exe"
"wininit.exe",2093,"C:\Windows\system32\wininit.exe"
"cmd.exe",1223,"C:\Windows\System32\cmd.exe"
"chrome.exe",5321,"C:\Users\user\AppData\Local\Google\Chrome\Application\chrome.exe"
"dellupdate.exe",1523,"C:\Program Files (x86)\Dell Update\Update.exe"
"systemd.exe",2374,"C:\Windows\System32\svchost.exe"

.  Use the "Windows10-baseline-processes.csv" file as the baseline.

Create a directory structure:

`
baseline.exe
Windows10-baseline-processes.csv
----+analyze_hosts\
------hostname1.csv
`

Then run:

`.\baseline.exe -d analyse_hosts -b Windows10-baseline-processes.csv -header "Path"`

That will print any deviations from the baseline file (Windows10-baseline-processescsv), which will help you identify any anomalies on your existing IIS server (hostname1).

`
[1.00 C:\Users\user\AppData\Local\Google\Chrome\Application\chrome.exe]
[1.00 C:\Windows\explorer.exe]
[1.00 C:\Program Files\Spotify\Spotify.exe]
[1.00 C:\Windows\system32\wininit.exe]
[1.00 C:\Windows\System32\cmd.exe]
[1.00 C:\Windows\System32\svchost.exe]
[1.00 C:\Windows\system32\taskeng.exe]
[1.00 C:\Program Files (x86)\Dell Update\Update.exe]
[0.50 C:\ProgramData\Dwm.exe]

`
In the above output, the "Dwm.exe" Path stands out because it doesn't exist in the baseline. It is also an anomaly because the Dwm.exe process is usually "dwm.exe" and is located under "C:\Windows\System32\" directory. The average of 0.50 means it is running on half of the 2 systems or just 1 system.

Based on the CSV file, you can select any header and filter on it. As long as the CSV file has the value passed to "-header" it will use it for it's calculations. For the `baseline` tool, it is designed to be used with similarly configured systems. The Windows 11 desktops of your developers may be different than those configured for your Human Resources personnel so be sure the baseline file is similar to the hosts being analyzed. The more similar the systems, the easier it will be to filter out anomalies.

### Logarithms

You will need to tune the threshold with the tools `mel` and `meeb` which uses a logarithm with a threshold of -3, by default. The logarithm is used because it can compress large datasets and help rare events to stand out. The term "compress" in this context means to group common data points. A value of -3 provides a good balance of whether or not an anomaly will stand out. You can change the value with the `-t` option.

Changing the values will help you identify anomalies.

#### Heterogenous Dataset

If the dataset is heterogenous, then you'll likely need a higher threshold (thus less sensitivity) to test it and start at around -2 or -2.5. That is because there is more variablity in the data so less sensitivity will lead to anomalies standing out easier.

The context is if you are analzying processes, for example, across different operating system versions or roles of the OS are different (eg. IIS web servers versus Exchange server). Not recommended, however.

#### Homogenous Dataset

If you have a similar dataset, then you'd want a lower threshold so the -3 is a good start.

Here is `mel` running on a homogenous dataset of 200 Windows 11 desktop systems with the default threshold of -3.

`
meeb.exe -d ../meebs/csvs/

[2 -3.838629 C:\ProgramData\system32\csrss.exe]
[1 -4.139659 C:\Program Files (x86)\iPod\ iTunes AppleTunes.exe]
`

It is able to find 2 anomalies. The first value is the number of hosts the file is on and then the logarithmic value. Since the dataset is homogenous, the lower threshold shows rare processes running.

#### Dataset size

Larger datasets provide higher confidence in detecting anomalies so a lower threshold works well (-3, -3.5, -4)

Smaller datasets have less confidence so a higher threshold is necessary -2 or -1 (for very small datasets).

Using the dataset from above which contains CSV files for 200 hosts, when we change the threshold to -2 it doesn't produce any differences. Changing it to -1 shows different results:

`
[snipped for brevity]
[200 -1.838629 C:\ProgramData\Ashampoo Winzip AshampooWinZip.exe]
[200 -1.838629 C:\Program Files\WebServer_11.24.0.0_x64__8wekyb3d8bbwe\WebServer.exe]
[200 -1.838629 C:\Program Files\SessionManager_11.24.0.0_x64__8wekyb3d8bbwe\SessionManager.exe]
[200 -1.838629 C:\ProgramData\Dell\Supportassist\DellSupportAssist.exe]
[200 -1.838629 C:\Program Files (x86)\Twitch Twitch.exe]
[200 -1.838629 C:\Program Files (x86)\Microsoft Edge\Application MicrosoftEdge.exe]
[200 -1.838629 C:\Windows\SystemHealth.exe]
[200 -1.838629 C:\Windows\system32\taskmgr.exe]
[199 -1.840806 C:\Program Files (x86)\iPod\ iTunes AppleiTunes.exe]
[193 -1.854101 C:\Windows\taskmgr.exe]
[2 -3.838629 C:\ProgramData\system32\csrss.exe]
[1 -4.139659 C:\Program Files (x86)\iPod\ iTunes AppleTunes.exe]
`
Note how there is a C:\Windows\taskmgr.exe on 193 hosts, but didn't show up as an anomaly with a threshold of -3. That is why you need to change the threshold so that you better spot anomalies that may exist outside of a given threshold. While larger datasets have higher confidence of anomalies with a lower threshold, some anomalies could still be missed. Accordingly, all of these factors are the reason you need to change the the threshold during your analysis. Also, the legit `taskmgr.exe` file is located in `C:\Windows\system32\taskmgr.exe`. If this was a real system, this would likely be a mass compromise UNLESS there is a custom program in that path or a third-party program had a similar name process. *CONTEXT! CONTEXT! CONTEXT!*

### mel and meeb

The one difference with `mel` and `meeb` is that `mel` prints the rows for the specified threshold, while `meeb` aggregates the header selected for analysis and prints the count of hosts where it appears and logarithmic value. The default threshold for the logarith is -3. When there is smaller dataset, the threshold will need to be higher such as -2.5 or -1.

The files are named for my friend and former colleague Dr. Melanie Brown at Champlain College who helped me with understanding that logarithms could be used for detecting anomalies. I think my new nickname for her is meeb.
