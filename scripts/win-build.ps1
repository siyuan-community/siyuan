echo 'pwsh -f ./scripts/build.ps1'

echo 'Building UI'

# nvm on
# nvm use 22

cd app

# $ENV:COREPACK_ENABLE_STRICT = '0'
$ENV:ELECTRON_MIRROR = 'https://npmmirror.com/mirrors/electron/'

pnpm install
pnpm run build

cd ..

## ----------------------------------------------------------------

echo 'Cleaning Builds'

Remove-Item './app/build' -Recurse
Remove-Item './app/kernel' -Recurse

echo 'Building Kernel'
# the C compiler "gcc" is necessary https://sourceforge.net/projects/mingw-w64/files/mingw-w64/

cd kernel

go version

$ENV:GO111MODULE = 'on'
$ENV:CGO_ENABLED = '1'

# $ENV:GOPROXY = 'https://goproxy.io'
# $ENV:GOPROXY = 'https://mirrors.aliyun.com/goproxy/'

go generate
goversioninfo -platform-specific=true -icon=resource/icon.ico -manifest=resource/goversioninfo.exe.manifest

$ENV:GOOS = 'windows'
$ENV:GOARCH = 'amd64'

go mod tidy
go build --tags fts5 -v -o "../app/kernel/SiYuan-Kernel.exe" -ldflags "-s -w -H=windowsgui" .

cd ..

## ----------------------------------------------------------------

echo 'Building Electron'

cd app
pnpm run dist

# pause
