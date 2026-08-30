# Launch all backends and reverse proxy in separate windows from PowerShell
Set-Location -Path $PSScriptRoot

Get-ChildItem -Path "backends" -Filter "*.go" | ForEach-Object {
    $name = $_.BaseName
    $filePath = $_.FullName
    Write-Host "[STARTING] $name..." -ForegroundColor Green
    Start-Process cmd.exe -ArgumentList "/k title $name && go run `"$filePath`""
}

Start-Sleep -Seconds 1
Write-Host "[STARTING] Reverse Proxy..." -ForegroundColor Cyan
Start-Process cmd.exe -ArgumentList "/k title Reverse Proxy && go run main.go"

Write-Host "All backends and reverse proxy launched!" -ForegroundColor Cyan
