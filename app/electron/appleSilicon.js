const prereleasePattern = /-(?:alpha|beta|rc)(?:[.-]|\d|$)/i;

const getAppleSiliconDownloadURL = (version) => {
    const packageName = `siyuan-CE-v${version}-mac-arm64.dmg`;
    if (prereleasePattern.test(version)) {
        return `https://github.com/siyuan-community/siyuan/releases/download/v${version}/${packageName}`;
    }
    return `https://github.com/siyuan-community/siyuan/releases/download/v${version}/${packageName}`;
};

const shouldDownloadAppleSilicon = (response) => response === 0;

const shouldShowAppleSiliconWarning = ({
    isDevelopment,
    isPackaged,
    platform,
    runningUnderARM64Translation,
    simulateRosetta,
}) => {
    if (!isPackaged && isDevelopment && simulateRosetta) {
        return true;
    }
    return isPackaged && platform === "darwin" && runningUnderARM64Translation;
};

module.exports = {
    getAppleSiliconDownloadURL,
    shouldDownloadAppleSilicon,
    shouldShowAppleSiliconWarning,
};
