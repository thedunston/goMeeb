
<# 

This script will collect:

1. Loaded modules for each process.
2. Local and Domain users.
3. Running Services
4. Scheduled Tasks
5. Running Processes

Be sure to test that it works before putting into production. EDRs may flag it or PowerShell scripts may disabled.

Try with: powershell.exe -ExecutionPolicy Bypass -Scope Current_user -File "CollectData.ps1"

NOTE: Look for this type of "Bypass" in your SIEM with powershell because attackers do the same thing.

Each of the CSV files generate different data Headers.

When you perform an analysis, select one of the datasets such as Scheduled tasks.

Then select the header that you want to perform the analysis on.

One suggestion is to select the ProcessName header first and check for any anomalies. (meeb tool)

Then, print the entire dataset, which only displays unique fields being sorted such as the FullPath and the numer of times it appearrs. (mel tool)

Generally, the higher the count number, the more likely it is a normal process, DLL, etc.

The lowest values are the ones to start the analysis on.

When collecting data in an enterprise, perform the task in phases until there is full coverage. That will help uncover bugs in the process. While
no performance impacts are expected, it will depend on the system and its unique context. Minimum, start by collecting on a subset of a group of
systems. If you have 5 web servers, collect from all, if you have 20, collect from 10 percent, if you have more than 100 of similar systems, collect
from 20% to start, etc.  Regardless, start small and slowly scale. It is not recommend to try and run this on hundreds of systems or two systems
without monitoring for performance impacts.

A sample of the data output is provided in case you have existing tools, like your SIEMs, that can output data in a similar format or can perform
a logarithmic (base 10) search on a dataset.

Unrelated: Install Sysmon. :)  For Linux, install auditd.

#>

# Change the storage location for the CSV files with the trailing slash.
$storePath = "c:\users\public\"

if (Test-Path -Path $storePath) {

    $hostname = $env:COMPUTERNAME

    # Add the hostname as a prefix so it 
    $saveFile = "$storePath\$hostname"

} else {
    
    "Path doesn't exist."
    Exit

}

######## --------------------

<#

Get Loaded Modules

This process may take a minute or so to run depending on the number of processes on the system
since it has to extract the loaded modules for each one.

Sample output:
"Hostname","ProcessName","ProcessID","ModuleName","FullPath"
"DPC","AggregatorHost","9556","AggregatorHost.exe","C:\WINDOWS\System32\AggregatorHost.exe"
"DPC","AggregatorHost","9556","ntdll.dll","C:\WINDOWS\SYSTEM32\ntdll.dll"
"DPC","AggregatorHost","9556","KERNEL32.DLL","C:\WINDOWS\System32\KERNEL32.DLL"
"DPC","AggregatorHost","9556","KERNELBASE.dll","C:\WINDOWS\System32\KERNELBASE.dll"
"DPC","AggregatorHost","9556","msvcp_win.dll","C:\WINDOWS\System32\msvcp_win.dll"
"DPC","AggregatorHost","9556","ucrtbase.dll","C:\WINDOWS\System32\ucrtbase.dll"
"DPC","AggregatorHost","9556","RPCRT4.dll","C:\WINDOWS\System32\RPCRT4.dll"

#>

# Get all processes.
$allProcesses = Get-Process | Where-Object { $_.Modules.Count -gt 0 }

# Array for holding the modules.
$moduleList = @()


# Loop through each process and obtain the modules loaded with it.
foreach ($process in $allProcesses) {
    
    try {

        # The format is the header of the CSV file is on the left and the value on the right.
        $process.Modules | ForEach-Object {
            $moduleList += [PSCustomObject]@{
            
                "HostName"    = $hostname
                "ProcessName" = $process.ProcessName
                "ProcessID"   = $process.Id
                "ModuleName"  = $_.ModuleName
                "FullPath"    = $_.FileName

            }

        }

    } catch {

        # The program will continue running if there is an error with a process.
        Write-Host "Failed to retrieve modules for process: $($process.ProcessName)"
    
    }

}

# Export to CSV. Based on the language of your OS, the UTF8 may need to be changed.
$moduleList | Export-Csv -Path $saveFile"_LoadedModules.csv" -NoTypeInformation -Encoding UTF8


<#


Retrieves a list of all local and domain users on the system.It may take a minute to run depending on the number of users.

Sample output:
"Hostname","UserName","FullName","Domain","AccountStatus","LastLogon"
"DPC","adm1n","","dpc","Enabled",
"DPC","josephus","","dpc","Enabled",
"DPC","patrick","patrick","dpc","Enabled",
"DPC","pleebus","","dpc","Enabled",

#>

# Get the hostname.
$hostname = $env:COMPUTERNAME

# Create an array to hold the user details.
$userDetailsList = @()

# Get all local users. The filter is to separate them from the domain users.
$localUsers = Get-WmiObject -Class Win32_UserAccount -Filter "LocalAccount=True" 

# Check if the machine is part of a domain.
$domainInfo = (Get-WmiObject -Class Win32_ComputerSystem).Domain
$domainUsers = @()

# Basic check to see if the computer is not on a domain.
# This may need to be changed from "WORKGROUP" to your specific environment.
if ($domainInfo -ne "WORKGROUP") {

    # Get all domain users. The filter is to separate them from the local user accounts.
    $domainUsers = Get-WmiObject -Class Win32_UserAccount -Filter "LocalAccount=False"
}

# Combine local and domain users
$allUsers = $localUsers + $domainUsers

# Process each user and retrieve detailed information.
foreach ($user in $allUsers) {

    try {

        # Create a custom object with the user details
        $userDetails = [PSCustomObject]@{
            "Hostname"      = $hostname
            "UserName"      = $user.Name
            "FullName"      = $user.FullName
            "Domain"        = $user.Domain
            "AccountStatus" = if ($user.Disabled) { "Disabled" } else { "Enabled" }
           
        }

        $userDetailsList += $userDetails

    } catch {

        Write-Host "Failed to retrieve details for user: $($user.Name)"
    }

}

# Export to CSV. Based on the language of your OS, the UTF8 may need to be changed..
$userDetailsList | Export-Csv -Path $saveFile"_UserDetails.csv" -NoTypeInformation -Encoding UTF8

<#

Get Scheduled Tasks and their details.

"Hostname","TaskName","TaskPath","TaskState","LastRunTime","NextRunTime","LastTaskResult","Triggers","Actions","Author","Description"
"DPC","StartupAppTask","\Microsoft\Windows\Application Experience\",,"8/12/2024 10:39:39 PM",,"0",,"%windir%\system32\rundll32.exe",,"Scans startup entries and raises notification to the
user if there are too many startup entries."
"DPC","appuriverifierdaily","\Microsoft\Windows\ApplicationData\",,"8/14/2024 12:18:18 AM",,"0",,"%windir%\system32\AppHostRegistrationVerifier.exe",,"Verifies AppUriHandler host registrations."
"DPC","appuriverifierinstall","\Microsoft\Windows\ApplicationData\",,"8/9/2024 3:48:48 PM",,"0",,"%windir%\system32\AppHostRegistrationVerifier.exe",,"Verifies AppUriHandler host registrations."
"DPC","CleanupTemporaryState","\Microsoft\Windows\ApplicationData\",,"8/9/2024 11:15:15 PM",,"0",,"%windir%\system32\rundll32.exe","SYSTEM","Cleans up each package's unused temporary files."

#>

# Get all scheduled tasks.
$scheduledTasks = Get-ScheduledTask

# Create an array to hold detailed information about each task.
$taskDetailsList = @()

foreach ($task in $scheduledTasks) {

    try {

        # Task name with path.
        $taskFullName = $task.TaskPath + $task.TaskName
        
        $taskInfo = Get-ScheduledTaskInfo -TaskPath $task.TaskPath -TaskName $task.TaskName
        
        $taskDetails = [PSCustomObject]@{

            "Hostname"        = $hostname
            "TaskName"       = $task.TaskName
            "TaskPath"       = $task.TaskPath
            "TaskState"      = $taskInfo.State
            "LastRunTime"   = $taskInfo.LastRunTime
            "NextRunTime"   = $taskInfo.NextRunTime
            "LastTaskResult"= $taskInfo.LastTaskResult
            "Triggers"        = ($task.Triggers | ForEach-Object { $_.At })
            "Actions"         = ($task.Actions | ForEach-Object { $_.Execute })
            "Author"          = $task.Principal.UserId
            "Description"     = $task.Description
        }

        $taskDetailsList += $taskDetails
    } catch {
        Write-Host "Failed to retrieve details for task: $($task.TaskPath)$($task.TaskName)"
    }
}

# Export to CSV. Based on the language of your OS, the UTF8 may need to be changed.
$taskDetailsList | Export-Csv -Path $saveFile"_ScheduledTasksDetails.csv" -NoTypeInformation -Encoding UTF8

<#

Get Registered Services

Sample Output
"DPC","ServiceName","DisplayName","Status","ProcessID","ProcessName","ExecutablePath"
"DPC","AarSvc_c38c6","AarSvc_c38c6","Stopped","0","Idle","C:\WINDOWS\system32\svchost.exe -k AarSvcGroup -p"
"DPC","AJRouter","AllJoyn Router Service","Stopped","0","Idle","C:\WINDOWS\system32\svchost.exe -k LocalServiceNetworkRestricted -p"
"DPC","ALG","Application Layer Gateway Service","Stopped","0","Idle","C:\WINDOWS\System32\alg.exe"
"DPC","AppIDSvc","Application Identity","Stopped","0","Idle","C:\WINDOWS\system32\svchost.exe -k LocalServiceNetworkRestricted -p"

#>

# Get all registered services.
$services = Get-Service

# Create an array to hold the service details.
$serviceDetailsList = @()

foreach ($service in $services) {
    # Get the service name
    $serviceName = $service.Name

    try {

        # Get the WMI object for the service to retrieve the path to the executable. This supports older Powershell
        # versions that don't print the path using Get-Service.
        $wmiService = Get-WmiObject Win32_Service -Filter "Name='$serviceName'"
        
        # Get the process associated with the service
        $process = Get-Process -Id $wmiService.ProcessId -ErrorAction SilentlyContinue
             
        $serviceDetails = [PSCustomObject]@{
            "Hostname" = $hostname
            "Service Name" = $serviceName
            "Display Name" = $service.DisplayName
            "Status"       = $service.Status
            "ProcessID"   = $wmiService.ProcessId
            "ProcessName" = $process.ProcessName
            "ExecutablePath" = $wmiService.PathName.Trim('"')
        }

        $serviceDetailsList += $serviceDetails
    } catch {
        Write-Host "Failed to retrieve details for service: $serviceName"
    }
}

# Export to CSV. Based on the language of your OS, the UTF8 may need to be changed.
$serviceDetailsList | Export-Csv -Path $saveFile"_RegisteredServicesDetails.csv" -NoTypeInformation -Encoding UTF8

<#

Get running processes.

"HostName","ProcessID","ProcessName","ExecutablePath","Username"
"DPC","9556","AggregatorHost","C:\WINDOWS\System32\AggregatorHost.exe","NT AUTHORITY\SYSTEM"
"DPC","7376","ApplicationFrameHost","C:\WINDOWS\system32\ApplicationFrameHost.exe","dpc\pleebus"
"DPC","52548","backgroundTaskHost","C:\WINDOWS\system32\backgroundTaskHost.exe","dpc\pleebus"
"DPC","66540","backgroundTaskHost","C:\WINDOWS\system32\backgroundTaskHost.exe","dpc\pleebus"
"DPC","69672","backgroundTaskHost","C:\WINDOWS\system32\backgroundTaskHost.exe","dpc\pleebus"
"DPC","1404","brave",,"dpc\pleebus"

#>

# Create an array to hold the process details.
$processDetailsList = @()

# Get all running processes.
$processes = Get-Process

foreach ($process in $processes) {

    try {

        # Get the user owning the process. This is an example where elevated privileges are required.
        # Otherwise, the user name will not show.
        $wmiProcess = Get-WmiObject Win32_Process -Filter "ProcessId='$($process.Id)'"
        $owner = $wmiProcess.GetOwner()
        $username = "$($owner.Domain)\$($owner.User)"

        $processDetails = [PSCustomObject]@{
            "HostName"     = $hostname
            "ProcessID"    = $process.Id
            "ProcessName"  = $process.ProcessName
            "ExecutablePath" = $process.Path
            "Username"     = $username
        }

        $processDetailsList += $processDetails

    } catch {
        
        Write-Host "Failed to retrieve details for process: $($process.ProcessName)"
    }
}

# Export to CSV. Based on the language of your OS, the UTF8 may need to be changed.
$processDetailsList | Export-Csv -Path $saveFile"_ProcessDetails.csv" -NoTypeInformation -Encoding UTF8
