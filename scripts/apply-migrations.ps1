param(
  [string]$ComposeService = "postgres",
  [string]$DbUser = "cbt",
  [string]$DbName = "cbt",
  [string]$MigrationsPath = "backend/migrations"
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path $MigrationsPath)) {
  throw "Migrations path not found: $MigrationsPath"
}

docker compose up -d $ComposeService | Out-Host

$ready = $false
for ($i = 0; $i -lt 60; $i++) {
  docker compose exec -T $ComposeService pg_isready -U $DbUser *> $null
  if ($LASTEXITCODE -eq 0) {
    $ready = $true
    break
  }
  Start-Sleep -Seconds 2
}
if (-not $ready) {
  throw "PostgreSQL is not ready"
}

$appliedRaw = docker compose exec -T $ComposeService psql -U $DbUser -d $DbName -At -c "SELECT version FROM schema_migrations ORDER BY version;"
$applied = @{}
foreach ($line in $appliedRaw) {
  $clean = "$line".Trim()
  if ($clean) {
    $applied[$clean] = $true
  }
}

$files = Get-ChildItem $MigrationsPath -Filter "*.sql" | Sort-Object Name
foreach ($file in $files) {
  $version = [System.IO.Path]::GetFileNameWithoutExtension($file.Name)
  if ($applied.ContainsKey($version)) {
    Write-Host "skip $version"
    continue
  }
  Write-Host "apply $version"
  Get-Content $file.FullName | docker compose exec -T $ComposeService psql -v ON_ERROR_STOP=1 -U $DbUser -d $DbName
}

Write-Host "migrations complete"
