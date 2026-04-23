$r = Invoke-WebRequest -Method OPTIONS `
  -Uri "https://data.ai-dia.com/api/v1/client/1893a372-f0e7-4b2e-bccd-99338c341c54/telegram/bot-token" `
  -Headers @{
    "Origin" = "https://ai-dia.com"
    "Access-Control-Request-Method" = "PATCH"
    "Access-Control-Request-Headers" = "content-type,authorization"
  } -UseBasicParsing

Write-Host "Status: $($r.StatusCode)"
Write-Host ""
Write-Host "=== Response Headers ==="
$r.Headers | Format-List
