[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)][string]$ProcessName,
    [int]$ProcessId = 0,
    [int]$WarmupSeconds = 120,
    [int]$Seconds = 300,
    [double]$Interval = 1,
    [string]$Out
)

$ErrorActionPreference = 'Stop'

$procs = @(Get-Process -Name $ProcessName -ErrorAction SilentlyContinue)
if ($ProcessId -gt 0) { $procs = @($procs | Where-Object { $_.Id -eq $ProcessId }) }
if ($procs.Count -eq 0) {
    [Console]::Error.WriteLine("process not found: $ProcessName")
    exit 2
}
if ($procs.Count -gt 1) {
    $ids = ($procs | ForEach-Object { $_.Id }) -join ','
    [Console]::Error.WriteLine("multiple '$ProcessName' processes ($ids); pass -ProcessId")
    exit 2
}
$p = $procs[0]
$procId = $p.Id

if (-not $Out) {
    $buildDir = Join-Path (Split-Path -Parent $PSScriptRoot) 'build'
    $Out = Join-Path $buildDir ("measure-{0}-{1}.csv" -f $ProcessName, (Get-Date -Format 'yyyyMMdd-HHmmss'))
}
$outDir = Split-Path -Parent ([IO.Path]::GetFullPath($Out))
if (-not (Test-Path -LiteralPath $outDir)) { New-Item -ItemType Directory -Path $outDir | Out-Null }

if ($WarmupSeconds -gt 0) { Start-Sleep -Seconds $WarmupSeconds }

$inv = [Globalization.CultureInfo]::InvariantCulture
$rows = New-Object System.Collections.Generic.List[object]
$exited = $false

try {
    $p.Refresh()
    if ($p.HasExited) { throw 'exited' }
    $prevCpu = $p.TotalProcessorTime.TotalSeconds
    $prevAt = [Diagnostics.Stopwatch]::GetTimestamp()
    $startAt = $prevAt
    $freq = [Diagnostics.Stopwatch]::Frequency
    $first = $true

    while ($true) {
        Start-Sleep -Milliseconds ([int]($Interval * 1000))
        $p.Refresh()
        if ($p.HasExited) { throw 'exited' }
        $now = [Diagnostics.Stopwatch]::GetTimestamp()
        $cpu = $p.TotalProcessorTime.TotalSeconds
        $wall = ($now - $prevAt) / $freq
        $elapsed = ($now - $startAt) / $freq
        $pct = if ($wall -gt 0) { ($cpu - $prevCpu) / $wall * 100 } else { 0 }
        $prevCpu = $cpu
        $prevAt = $now
        $rows.Add([pscustomobject]@{
            timestamp = (Get-Date).ToString('o')
            elapsed_s = [math]::Round($elapsed, 1)
            cpu_pct   = [math]::Round($pct, 2)
            ws_mb     = [math]::Round($p.WorkingSet64 / 1MB, 2)
            private_mb = [math]::Round($p.PrivateMemorySize64 / 1MB, 2)
            handles   = $p.HandleCount
            threads   = $p.Threads.Count
        })
        if ($elapsed -ge $Seconds) { break }
    }
} catch {
    if ($_.Exception.Message -eq 'exited') { $exited = $true } else { throw }
}

$rows | Export-Csv -LiteralPath $Out -NoTypeInformation -Encoding UTF8

$n = $rows.Count
if ($n -gt 0) {
    $cpuAvg = ($rows | Measure-Object cpu_pct -Average).Average
    $cpuMax = ($rows | Measure-Object cpu_pct -Maximum).Maximum
    $wsMax = ($rows | Measure-Object ws_mb -Maximum).Maximum
    $prMax = ($rows | Measure-Object private_mb -Maximum).Maximum
    $hS = $rows[0].handles; $hE = $rows[$n - 1].handles
    $tS = $rows[0].threads; $tE = $rows[$n - 1].threads
    $window = $rows[$n - 1].elapsed_s
} else {
    $cpuAvg = 0; $cpuMax = 0; $wsMax = 0; $prMax = 0
    $hS = 0; $hE = 0; $tS = 0; $tE = 0; $window = 0
}

function F($v) { ([double]$v).ToString('0.##', $inv) }

$summary = [ordered]@{
    process       = $ProcessName
    pid           = $procId
    warmup_s      = $WarmupSeconds
    window_s      = F $window
    samples       = $n
    cpu_avg_pct   = F $cpuAvg
    cpu_max_pct   = F $cpuMax
    ws_max_mb     = F $wsMax
    private_max_mb = F $prMax
    handles_start = $hS
    handles_end   = $hE
    handles_delta = $hE - $hS
    threads_start = $tS
    threads_end   = $tE
    threads_delta = $tE - $tS
    csv           = $Out
}
if ($exited) { $summary['exited_early'] = 1 }

($summary.GetEnumerator() | ForEach-Object { '{0,-15} {1}' -f $_.Key, $_.Value }) | Write-Output
($summary.GetEnumerator() | ForEach-Object { '{0}={1}' -f $_.Key, $_.Value }) -join ' ' | Write-Output

if ($exited) { exit 3 }
exit 0
