$ip = "192.168.1.202"
$port = 5005

Write-Host "=== 1. Cek IP komputer ini ===" -ForegroundColor Cyan
Get-NetIPAddress -AddressFamily IPv4 | Where-Object { $_.IPAddress -notlike "169.254*" -and $_.IPAddress -ne "127.0.0.1" } | Select-Object InterfaceAlias, IPAddress, PrefixLength | Format-Table -AutoSize

Write-Host "`n=== 2. Ping ke device ($ip) ===" -ForegroundColor Cyan
$ping = Test-Connection -ComputerName $ip -Count 4 -ErrorAction SilentlyContinue
if ($ping) {
    $ping | Select-Object Address, ResponseTime | Format-Table -AutoSize
    Write-Host "Ping BERHASIL - device hidup dan satu jaringan." -ForegroundColor Green
} else {
    Write-Host "Ping GAGAL - device tidak terjangkau dari PC ini. Cek kabel/wifi, atau device belum benar-benar dapat IP itu." -ForegroundColor Red
}

Write-Host "`n=== 3. Cek port $port (protokol Fingerspot) ===" -ForegroundColor Cyan
$tcp = Test-NetConnection -ComputerName $ip -Port $port -WarningAction SilentlyContinue
$tcp | Select-Object ComputerName, RemotePort, PingSucceeded, TcpTestSucceeded | Format-Table -AutoSize
if ($tcp.TcpTestSucceeded) {
    Write-Host "Port $port TERBUKA - device merespons di port ini. Masalahnya kemungkinan besar murni soal password/comm key, bukan jaringan." -ForegroundColor Green
} else {
    Write-Host "Port $port TERTUTUP/tidak merespons - coba juga cek port 4370 (default umum ZKTeco):" -ForegroundColor Yellow
    $tcp2 = Test-NetConnection -ComputerName $ip -Port 4370 -WarningAction SilentlyContinue
    $tcp2 | Select-Object ComputerName, RemotePort, TcpTestSucceeded | Format-Table -AutoSize
}

Write-Host "`n=== Selesai. Kirim seluruh hasil di atas untuk dianalisis. ===" -ForegroundColor Cyan
