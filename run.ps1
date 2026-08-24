# Launch all backends in separate CMD windows from PowerShell
Set-Location -Path $PSScriptRoot

Get-ChildItem -Path "backends" -Filter "*.go" | ForEach-Object {
    $name = $_.BaseName
    $filePath = $_.FullName
    Write-Host "[STARTING] $name..." -ForegroundColor Green
    Start-Process cmd.exe -ArgumentList "/k title $name && go run `"$filePath`""
}

Write-Host "All backends launched!" -ForegroundColor Cyan
