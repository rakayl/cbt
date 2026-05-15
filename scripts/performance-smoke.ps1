param(
  [string]$BaseUrl = "http://localhost",
  [string]$Email = "admin@example.edu",
  [string]$Password = "ChangeMe123!",
  [string]$TenantId = "10000000-0000-0000-0000-000000000001",
  [int]$Requests = 100,
  [int]$Concurrency = 10
)

$ErrorActionPreference = "Stop"

$login = Invoke-RestMethod -Method Post -Uri "$BaseUrl/api/v1/auth/login" -ContentType "application/json" -Body (@{
  email = $Email
  password = $Password
  tenant_id = $TenantId
  device_name = "performance-smoke"
  fingerprint = "performance-smoke"
} | ConvertTo-Json)

$headers = @{
  Authorization = "Bearer $($login.data.access_token)"
  "X-Tenant-ID" = $TenantId
}

$perWorker = [math]::Ceiling($Requests / [math]::Max(1, $Concurrency))
$workers = 1..$Concurrency | ForEach-Object {
  $workerRequests = [math]::Min($perWorker, [math]::Max(0, $Requests - (($_ - 1) * $perWorker)))
  Start-Job -ArgumentList $BaseUrl, $headers, $workerRequests -ScriptBlock {
    param($BaseUrl, $Headers, $WorkerRequests)
    $errors = @()
    $durations = @()
    for ($i = 0; $i -lt $WorkerRequests; $i++) {
      $sw = [System.Diagnostics.Stopwatch]::StartNew()
      try {
        Invoke-RestMethod -Method Get -Uri "$BaseUrl/api/v1/questions/?page=1&limit=5" -Headers $Headers | Out-Null
        $sw.Stop()
        $durations += $sw.Elapsed.TotalMilliseconds
      } catch {
        $sw.Stop()
        $errors += $_.Exception.Message
      }
    }
    [pscustomobject]@{ Durations = $durations; Errors = $errors }
  }
}

$results = $workers | Wait-Job | Receive-Job
$workers | Remove-Job

$durationArray = @($results | ForEach-Object { $_.Durations })
$errorArray = @($results | ForEach-Object { $_.Errors })
$avg = if ($durationArray.Count) { [math]::Round(($durationArray | Measure-Object -Average).Average, 2) } else { 0 }
$max = if ($durationArray.Count) { [math]::Round(($durationArray | Measure-Object -Maximum).Maximum, 2) } else { 0 }

Write-Host "requests=$Requests concurrency=$Concurrency ok=$($durationArray.Count) errors=$($errorArray.Count) avg_ms=$avg max_ms=$max"
if ($errorArray.Count -gt 0) {
  $errorArray | Select-Object -First 5 | ForEach-Object { Write-Host "error: $_" }
  exit 1
}
