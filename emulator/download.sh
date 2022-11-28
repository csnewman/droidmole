#ARCH='arm64'
#THROTTLED='-throttled'
ARCH='x86_64'
THROTTLED=''

URL=https://ci.android.com/builds/latest/branches/aosp-master${THROTTLED}/targets/aosp_cf_${ARCH}_phone-userdebug/view/BUILD_INFO
RURL=$(curl -Ls -o /dev/null -w %{url_effective} ${URL})

echo "Downloading android"
IMG=aosp_cf_${ARCH}_phone-img-$(echo $RURL | awk -F\/ '{print $6}').zip
wget -nv ${RURL%/view/BUILD_INFO}/raw/${IMG}

echo "Downloading cuttlefish"
wget -nv ${RURL%/view/BUILD_INFO}/raw/cvd-host_package.tar.gz

echo "Creating device"
rm -r device/
mkdir device
cd device
unzip "../${IMG}"
tar xzvf ../cvd-host_package.tar.gz
