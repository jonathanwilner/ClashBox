param(
  [Parameter(Mandatory = $true)][string]$UnsignedHap,
  [Parameter(Mandatory = $true)][string]$OutputHap,
  [Parameter(Mandatory = $true)][string]$KeyAlias,
  [Parameter(Mandatory = $true)][string]$Certificate,
  [Parameter(Mandatory = $true)][string]$Profile,
  [Parameter(Mandatory = $true)][string]$Keystore,
  [string]$DevEcoHome = 'C:\Huawei\DevEcoStudio\DevEco Studio 26.0.0'
)

$ErrorActionPreference = 'Stop'
$keyPassword = $env:CLASHBOX_KEY_PASSWORD
$storePassword = $env:CLASHBOX_STORE_PASSWORD
if ([string]::IsNullOrEmpty($keyPassword) -or [string]::IsNullOrEmpty($storePassword)) {
  throw 'Set CLASHBOX_KEY_PASSWORD and CLASHBOX_STORE_PASSWORD from the local secret store.'
}

$java = Join-Path $DevEcoHome 'jbr\bin\java.exe'
$signer = Join-Path $DevEcoHome 'sdk\default\openharmony\toolchains\lib\hap-sign-tool.jar'
foreach ($path in @($java, $signer, $UnsignedHap, $Certificate, $Profile, $Keystore)) {
  if (-not (Test-Path -LiteralPath $path)) { throw "Required input not found: $path" }
}

$outputDirectory = Split-Path -Parent $OutputHap
New-Item -ItemType Directory -Path $outputDirectory -Force | Out-Null

& $java -jar $signer sign-app -mode localSign -keyAlias $KeyAlias `
  -keyPwd $keyPassword -appCertFile $Certificate -profileFile $Profile `
  -profileSigned 1 -inFile $UnsignedHap -signAlg SHA256withECDSA `
  -keystoreFile $Keystore -keystorePwd $storePassword -outFile $OutputHap `
  -compatibleVersion 23 -signCode 1
if ($LASTEXITCODE -ne 0) { throw "HAP signing failed with exit code $LASTEXITCODE" }

$verifyRoot = Join-Path ([System.IO.Path]::GetTempPath()) ('clashbox-jw-verify-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $verifyRoot | Out-Null
try {
  & $java -jar $signer verify-app -inFile $OutputHap `
    -outCertChain (Join-Path $verifyRoot 'certchain.cer') `
    -outProfile (Join-Path $verifyRoot 'profile.p7b')
  if ($LASTEXITCODE -ne 0) { throw "HAP verification failed with exit code $LASTEXITCODE" }
  Get-FileHash -Algorithm SHA256 -LiteralPath $OutputHap | Format-List
} finally {
  Remove-Item -LiteralPath $verifyRoot -Recurse -Force -ErrorAction SilentlyContinue
}
