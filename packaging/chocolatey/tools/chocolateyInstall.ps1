$ErrorActionPreference = 'Stop'
$version = '0.1.0'
$asset = "martis_${version}_windows_amd64.zip"
$base = "https://github.com/wahyuakbarwibowo/martis/releases/download/v${version}"
$checksumPath = Join-Path $env:TEMP 'martis-checksums.txt'
$archivePath = Join-Path $env:TEMP $asset
Invoke-WebRequest -Uri "$base/checksums.txt" -OutFile $checksumPath
$line = Get-Content $checksumPath | Where-Object { $_ -match "\s+$([regex]::Escape($asset))$" } | Select-Object -First 1
if (-not $line) { throw "Checksum entry for $asset not found" }
$expected = ($line -split '\s+')[0].ToLowerInvariant()
Get-ChocolateyWebFile -PackageName 'martis' -FileFullPath $archivePath -Url "$base/$asset"
$actual = (Get-FileHash -Path $archivePath -Algorithm SHA256).Hash.ToLowerInvariant()
if ($actual -ne $expected) { throw "SHA256 mismatch for $asset" }
$toolsDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
Get-ChocolateyUnzip -FileFullPath $archivePath -Destination $toolsDir
Install-BinFile -Name 'martis' -Path (Join-Path $toolsDir 'martis.exe')
