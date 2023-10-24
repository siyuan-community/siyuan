echo 'pwsh -f ./scripts/build.ps1'

echo 'Building UI'

nvm on
nvm use 20

cd app

pnpm install
pnpm run build

cd ..

echo 'Cleaning Builds'

Remove-Item './app/build' -Recurse
Remove-Item './app/kernel' -Recurse

echo 'Building Kernel'
# the C compiler "gcc" is necessary https://sourceforge.net/projects/mingw-w64/files/mingw-w64/

cd kernel

go version

$ENV:GO111MODULE = 'on'
$ENV:GOPROXY = 'https://goproxy.io'
$ENV:CGO_ENABLED = '1'

$ENV:GOOS = 'windows'
$ENV:GOARCH = 'amd64'

go generate
goversioninfo -platform-specific=true -icon=resource/icon.ico -manifest=resource/goversioninfo.exe.manifest

go mod tidy
go build --tags fts5 -v -o "../app/kernel/SiYuan-Kernel.exe" -ldflags "-s -w -H=windowsgui" .

cd ..

echo 'Building Electron'

cd app
pnpm run dist

# pause
