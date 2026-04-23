$uri = "https://data.ai-dia.com/api/v1/client/1893a372-f0e7-4b2e-bccd-99338c341c54/telegram/bot-token"
$token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkYXRhIjp7ImVtYWlsIjoiaGFmaXpAZXhhbXBsZS5jb20iLCJuYW1lIjoiSGFmaXoiLCJyb2xlIjoiQ2xpZW50IiwidGVuYW50X3NjaGVtYSI6ImhhZml6IiwidXNlcl9pZCI6IjE4OTNhMzcyLWYwZTctNGIyZS1iY2NkLTk5MzM4YzM0MWM1NCIsInVzZXJuYW1lIjoiaGFmaXoifSwiZXhwIjoxNzc1NzQxNzAyLCJpYXQiOjE3NzU2NTUzMDJ9.og66bdMnmveDxxqSO0nvjN0FfVRWgYAcmk-jbkxIzHQ"
$body = '{"bot_token":"8203532707:AAGWKY9so3Jg7jod54P41chJ7VeQkKrxkUQ"}'
$headers = @{
    "Authorization" = "Bearer $token"
    "Content-Type"  = "application/json"
}

Invoke-RestMethod -Method PATCH -Uri $uri -Headers $headers -Body $body -ContentType "application/json"
