# scanpi
scanpi - Web interface for SANE (Scanner Access Now Easy)

Scan documents in your local network with a Raspberry Pi.
Scanpi allows your devices at home to access your old scanner.

![Jobs view](scanpi-jobs.jpg)
![Job view](scanpi-job.jpg)

## Build

The build process supports cross platform compilation.
To set the target architecture add the argument `ARCH` with the value when calling `make`.
Supported architectures:
* amd64
* i386
* armhf
* arm64

```
make ARCH=armhf
```    

## Install

After building the package you can install the package using `dpkg` command.

    # dpkg -i scanpi_1.0.0_armhf.deb

The service starts automatically after installing. The service will start after reboot.

## Configuration

The service can be configured modifying the file `/etc/opt/scanpi.conf`.
You have to restart the service to apply the new configuration.

## Service

You can control the service using `systemd`.
To start the service:

    # systemctl start scanpi

To stop the service:

    # systemctl stop scanpi

To restart the service:

    # systemctl restart scanpi
