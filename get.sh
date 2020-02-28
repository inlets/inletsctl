#!/bin/bash
# This script was adapted from https://github.com/openfaas/cli.openfaas.com/blob/master/get.sh

export OWNER=inlets
export REPO=inletsctl
export SUCCESS_CMD="$REPO version"
export BINLOCATION="/usr/local/bin"

version=$(curl -sI https://github.com/$OWNER/$REPO/releases/latest | grep -i location | awk -F"/" '{ printf "%s", $NF }' | tr -d '\r')

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

getPackage() {
    uname=$(uname)
    userid=$(id -u)

    suffix=""
    case $uname in
    "Darwin")
    suffix="-darwin.tgz"
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

    if [ -e $targetFile ]; then
        rm $targetFile
    fi

    url=https://github.com/$OWNER/$REPO/releases/download/$version/$REPO$suffix
    echo "Downloading package $url as $targetFile"

    curl -sSLf $url --output $targetFile

    if [ $? -ne 0 ]; then
        echo "Download Failed!"
        exit 1
    else
        extractFolder=$(echo $targetFile | sed "s/${REPO}${suffix}//g")
        echo "Download Complete, extracting $targetFile to $extractFolder ..."
        tar -xzf $targetFile -C $extractFolder
    fi

    if [ $? -ne 0 ]; then
        echo "\nFailed to expand archve: $targetFile"
        exit 1
    else
        # Remove the tar file
        echo "OK"
        rm $targetFile

        # Get the parent dir of the 'bin' folder holding the binary
        targetFile=$(echo $targetFile | sed "s+/${REPO}${suffix}++g")
        suffix=$(echo $suffix | sed 's/.tgz//g')

        targetFile="${targetFile}/bin/${REPO}${suffix}"

        chmod +x $targetFile

        # Calculate SHA
        shaurl=$(echo $url | sed 's/.tgz/.sha256/g')
        SHA256=$(curl -sLS $shaurl | awk '{print $1}')
        echo "SHA256 fetched from release: $SHA256"
        # NOTE to other maintainers
        # There needs to be two spaces between the SHA and the file in the echo statement
        # for shasum to compare the checksums
        echo "$SHA256  $targetFile" | shasum -a 256 -c -s

        if [ $? -ne 0 ]; then
            echo "SHA mismatch! This means there must be a problem with the download"
            exit 1
        else
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

                mv $targetFile $BINLOCATION/$REPO

                if [ "$?" = "0" ]; then
                    echo "New version of $REPO installed to $BINLOCATION"
                fi

                if [ -e $targetFile ]; then
                    rm $targetFile
                fi

            ${SUCCESS_CMD}
            fi
        fi
    fi
}

hasCli
getPackage

