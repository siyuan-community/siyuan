#!/usr/bin/env nu

print 'Usage: nu scripts/darwin-build.nu [--target <target>]'
print 'Options:'
print '  --target <target>  Build target: amd64, arm64, or all (default: all)'
print ''

def main [
    --target: string = 'all'  # Build target: amd64, arm64, or all
] {
    let INITIAL_DIR: string = (pwd)
    let SCRIPT_DIR: string = $env.FILE_PWD
    let PROJECT_ROOT: string = ($SCRIPT_DIR | path dirname)

    if $target not-in ['amd64', 'arm64', 'all'] {
        print $"Error: Invalid target '($target)'"
        print 'Valid targets are: amd64, arm64, all'
        exit 1
    }

    print 'Cleaning Builds'
    rm -rf ($PROJECT_ROOT | path join 'app/build')
    rm -rf ($PROJECT_ROOT | path join 'app/kernel-darwin')
    rm -rf ($PROJECT_ROOT | path join 'app/kernel-darwin-arm64')

    # ================ Build UI ================
    print ''
    print 'Building UI'
    cd ($PROJECT_ROOT | path join 'app')
    pnpm install
    pnpm run install:electron
    pnpm run build
    cd $PROJECT_ROOT

    # ================ Build Kernel ================
    print ''
    print 'Building Kernel'
    cd ($PROJECT_ROOT | path join 'kernel')
    go version

    $env.GO111MODULE = 'on'
    $env.GOPROXY = 'https://mirrors.aliyun.com/goproxy/,https://goproxy.cn,direct'
    $env.CGO_ENABLED = '1'
    $env.GOOS = 'darwin'

    if $target == 'amd64' or $target == 'all' {
        print ''
        print 'Building Kernel amd64'
        $env.GOARCH = 'amd64'
        go build -tags "fts5 sqlcipher" -o "../app/kernel-darwin/SiYuan-Kernel" -ldflags "-s -w" .
    }

    if $target == 'arm64' or $target == 'all' {
        print ''
        print 'Building Kernel arm64'
        $env.GOARCH = 'arm64'
        go build -tags "fts5 sqlcipher" -o "../app/kernel-darwin-arm64/SiYuan-Kernel" -ldflags "-s -w" .
    }
    cd $PROJECT_ROOT

    # ================ Build Electron App ================
    print ''
    print 'Building Electron App'
    cd ($PROJECT_ROOT | path join 'app')

    if $target == 'amd64' or $target == 'all' {
        print ''
        print 'Building Electron App amd64'
        pnpm run dist-darwin
    }

    if $target == 'arm64' or $target == 'all' {
        print ''
        print 'Building Electron App arm64'
        pnpm run dist-darwin-arm64
    }
    cd $PROJECT_ROOT

    print ''
    print '=============================='
    print '      Build successful!'
    print '=============================='

    cd $INITIAL_DIR
}
