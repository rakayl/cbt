param(
  [string]$BaseUrl = "http://localhost",
  [string]$Email = "admin@example.edu",
  [string]$Password = "ChangeMe123!",
  [string]$TenantId = "10000000-0000-0000-0000-000000000001"
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
    $params.Body = ($Body | ConvertTo-Json -Depth 30)
  }
  Invoke-RestMethod @params
}

Write-Host "health"
Invoke-Json GET "/healthz" | Out-Null

Write-Host "login admin"
$login = Invoke-Json POST "/api/v1/auth/login" @{
  email = $Email
  password = $Password
  tenant_id = $TenantId
  device_name = "e2e-smoke"
  fingerprint = "e2e-smoke"
}
$headers = @{
  Authorization = "Bearer $($login.data.access_token)"
  "X-Tenant-ID" = $TenantId
}

Write-Host "question usage"
$questionList = Invoke-Json GET "/api/v1/questions/?page=1&limit=1" $null $headers
if ($questionList.data.items.Count -gt 0) {
  $questionId = $questionList.data.items[0].id
  Invoke-Json GET "/api/v1/questions/$questionId/usage" $null $headers | Out-Null
}

Write-Host "published exam lock"
$examList = Invoke-Json GET "/api/v1/exams/?page=1&limit=20" $null $headers
$published = @($examList.data.items | Where-Object { $_.status -eq "published" } | Select-Object -First 1)
if ($published.Count -gt 0) {
  $exam = $published[0]
  try {
    Invoke-Json PUT "/api/v1/exams/$($exam.id)" @{
      code = $exam.code
      name = "$($exam.name) Locked Smoke"
      description = $exam.description
      status = "published"
      metadata = $exam.metadata
    } $headers | Out-Null
    throw "published exam update should have been rejected"
  } catch {
    if ($_.Exception.Message -like "*should have been rejected*") {
      throw
    }
    Write-Host "published exam update rejected as expected"
  }
}

Write-Host "proctoring timeline"
$session = docker compose exec -T postgres psql -U cbt -d cbt -At -c "SELECT id FROM exam_sessions WHERE tenant_id='$TenantId' AND deleted_at IS NULL ORDER BY created_at DESC LIMIT 1;"
$sessionId = "$session".Trim()
if ($sessionId) {
  Invoke-Json GET "/api/v1/proctoring/sessions/$sessionId/timeline" $null $headers | Out-Null
}

Write-Host "e2e smoke passed"
