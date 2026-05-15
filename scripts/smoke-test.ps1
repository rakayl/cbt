param(
  [string]$BaseUrl = "http://localhost",
  [string]$Email = "admin@example.edu",
  [string]$Password = "ChangeMe123!"
)

$ErrorActionPreference = "Stop"

function Invoke-Json($Method, $Path, $Body = $null, $Headers = @{}) {
  $params = @{
    Method = $Method
    Uri = "$BaseUrl$Path"
    Headers = $Headers
    ContentType = "application/json"
  }
  if ($null -ne $Body) {
    $params.Body = ($Body | ConvertTo-Json -Depth 20)
  }
  Invoke-RestMethod @params
}

Write-Host "Checking health..."
Invoke-Json GET "/healthz" | Out-Null

Write-Host "Logging in..."
$login = Invoke-Json POST "/api/v1/auth/login" @{
  email = $Email
  password = $Password
  device_name = "smoke-test"
  fingerprint = "smoke-test"
}

$token = $login.data.access_token
$tenant = $login.data.tenant_id
$headers = @{
  Authorization = "Bearer $token"
  "X-Tenant-ID" = $tenant
}

Write-Host "Checking protected endpoint..."
Invoke-Json GET "/api/v1/exam-sessions?page=1&limit=1" $null $headers | Out-Null

Write-Host "Smoke test passed."
