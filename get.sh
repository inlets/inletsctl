#!/bin/bash
# This script was adapted from https://github.com/openfaas/cli.openfaas.com/blob/master/get.sh

export VERIFY_CHECKSUM=1
export OWNER=inlets
export REPO=inletsctl
export BINLOCATION="/usr/local/bin"
export SUCCESS_CMD="$BINLOCATION/$REPO version"

version=$(curl -sI https://github.com/$OWNER/$REPO/releases/latest | grep -i "location:" | awk -F"/" '{ printf "%s", $NF }' | tr -d '\r')

if [ ! $version ]; then
    echo "Failed while attempting to install $REPO. Please manually install:"
    echo ""
    echo "1. Open your web browser and go to https://github.com/$OWNER/$REPO/releases"
    echo "2. Download the latest release for your platform. Call it '$REPO'."
    echo "3. chmod +x ./$REPO"
    echo "4. mv ./$REPO $BINLOCATION"
    exit 1
fi

hasCli() {

    hasCurl=$(which curl)
    if [ "$?" = "1" ]; then
        echo "You need curl to use this script."
        exit 1
    fi
}

checkHash(){
    
    sha_cmd="sha256sum"

    if [ ! -x "$(command -v $sha_cmd)" ]; then
        sha_cmd="shasum -a 256"
    fi

    if [ -x "$(command -v $sha_cmd)" ]; then
        
        targetFileDir=$(dirname $targetFile)
        
        (cd $targetFileDir && curl -sSL ${url%.*}.sha256|$sha_cmd -c >/dev/null)
   
        if [ "$?" != "0" ]; then
            rm $targetFile
            echo "Binary checksum didn't match. Exiting"
            exit 1
        fi   
    fi
}

getPackage() {
    uname=$(uname)
    userid=$(id -u)
    arch=$(uname -m)

    suffix=""
    case $uname in
    "Darwin")
        case $arch in
        "x86_64")
        suffix="-darwin.tgz"
        ;;
        esac
        case $arch in
        "arm64")
        suffix="-darwin-arm64.tgz"
        ;;
        esac
    ;;
    "Linux")
        arch=$(uname -m)
        echo $arch
        case $arch in
        "x86_64")
        suffix=".tgz"
        ;;
        esac
        case $arch in
        "aarch64")
        suffix="-arm64.tgz"
        ;;
        esac
        case $arch in
        "armv6l" | "armv7l")
        suffix="-armhf.tgz"
        ;;
        esac
    ;;
    esac

    targetFile="/tmp/$REPO$suffix"

    if [ "$userid" != "0" ]; then
        targetFile="$(pwd)/$REPO$suffix"
    fi

    if [ -e "$targetFile" ]; then
        rm "$targetFile"
    fi

    url=https://github.com/$OWNER/$REPO/releases/download/$version/$REPO$suffix
    echo "Downloading package $url as $targetFile"

    curl -sSLf $url --output "$targetFile"

    if [ $? -ne 0 ]; then
        echo "Download Failed!"
        exit 1
    else
        echo "Download complete, extracting $REPO from ${targetFile}..."
        tar -xzf "${targetFile}" -C "$(dirname ${targetFile})"
    fi

    if [ $? -ne 0 ]; then
        echo "\nFailed to expand archive: $targetFile"
        exit 1
    else
        # Remove the tar file
        echo "Binary extracted"
        rm "$targetFile"
        # Binary now extracted so remove the archive file extension
        targetFile=${targetFile%.*}

        if [ "$VERIFY_CHECKSUM" = "1" ]; then
            checkHash
        fi

        chmod +x "$targetFile"

            if [ ! -w "$BINLOCATION" ]; then
                echo
                echo "============================================================"
                echo "  The script was run as a user who is unable to write"
                echo "  to $BINLOCATION. To complete the installation the"
                echo "  following commands may need to be run manually."
                echo "============================================================"
                echo
                echo "  sudo cp $REPO$suffix $BINLOCATION/$REPO"
                echo
                ./$REPO$suffix version
            else

                echo
                echo "Running with sufficient permissions to attempt to move $REPO to $BINLOCATION"

                mv "$targetFile" $BINLOCATION/$REPO

                if [ "$?" = "0" ]; then
                    echo "New version of $REPO installed to $BINLOCATION"
                fi

                if [ -e "$targetFile" ]; then
                    rm "$targetFile"
                fi

            ${SUCCESS_CMD}
            fi
    fi
}

hasCli
getPackage

