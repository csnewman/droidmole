# /etc/init.d/cuttlefish-host-resources start
# HOME=$PWD ./bin/launch_cvd -report_anonymous_usage_stats=n

docker stop dmcf-1
docker rm dmcf-1
docker run \
    --cgroupns=host \
    --name "dmcf-1" \
    -i \
    --privileged \
    -v /sys/fs/cgroup:/sys/fs/cgroup:rw \
    -v "$PWD/device":/device \
    dmcf
