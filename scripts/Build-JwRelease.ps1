param(
  [string]$DevEcoHome = 'C:\Huawei\DevEcoStudio\DevEco Studio 26.0.0',
  [string]$OhpmRegistry = 'https://repo.harmonyos.com/ohpm/'
)

$ErrorActionPreference = 'Stop'
$env:JAVA_TOOL_OPTIONS = '-Dfile.encoding=UTF-8 -Duser.language=zh -Duser.country=CN'
$env:npm_config_registry = 'https://registry.npmmirror.com'
$env:NPM_CONFIG_REGISTRY = 'https://registry.npmmirror.com'

$projectRoot = Split-Path -Parent $PSScriptRoot
$ohpm = Join-Path $DevEcoHome 'tools\ohpm\bin\ohpm.bat'
$hvigorw = Join-Path $DevEcoHome 'tools\hvigor\bin\hvigorw.bat'

foreach ($tool in @($ohpm, $hvigorw)) {
  if (-not (Test-Path -LiteralPath $tool)) {
    throw "Required DevEco tool not found: $tool"
  }
}

Push-Location $projectRoot
try {
  & $ohpm install --registry $OhpmRegistry
  if ($LASTEXITCODE -ne 0) { throw "ohpm install failed with exit code $LASTEXITCODE" }

  & $hvigorw clean --no-daemon
  if ($LASTEXITCODE -ne 0) { throw "Hvigor clean failed with exit code $LASTEXITCODE" }

  & $hvigorw assembleHap --mode project -p product=release -p buildMode=release --no-daemon --stacktrace
  if ($LASTEXITCODE -ne 0) { throw "Hvigor assembleHap failed with exit code $LASTEXITCODE" }

  $haps = Get-ChildItem -LiteralPath (Join-Path $projectRoot 'entry\build') -Recurse -File -Filter '*unsigned.hap' |
    Sort-Object LastWriteTime -Descending
  if (-not $haps) { throw 'Build completed but no unsigned entry HAP was found.' }
  $haps[0].FullName
} finally {
  Pop-Location
}
