//go:build linux || darwin
// +build linux darwin

package parse

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/containers/buildah/define"
	"github.com/container-orchestrated-devices/container-device-interface/pkg"
	"github.com/opencontainers/runc/libcontainer/devices"
)

//go:build linux || darwin
// +build linux darwin

package parse

func DeviceFromCDI(device string) (define.ContainerDevices, error) {
	var devs define.ContainerDevices

	cdiClient, err := cdi.NewClient()
	if err != nil {
		return nil, fmt.Errorf("creating CDI client: %w", err)
	}
	defer cdiClient.Close()

	cdiDevices, err := cdiClient.GetDevice(device)
	if err != nil {
		return nil, fmt.Errorf("getting CDI device %s: %w", device, err)
	}

	for _, cdiDevice := range cdiDevices {
		dev := define.BuildahDevice{
			Device:       devices.Device{
					Type: cdiDevice.Type, 
					Major: cdiDevice.Major, 
					Minor: cdiDevice.Minor, 
					FileMode: os.FileMode(cdiDevice.FileMode), 
					Uid: cdiDevice.UID, 
					Gid: cdiDevice.GID},
			Source:       cdiDevice.HostPath,
			Destination:  cdiDevice.ContainerPath,
			Permissions:  cdiDevice.Permissions,
		}
		devs = append(devs, dev)
	}

	return devs, nil
}

func DeviceFromPath(device string) (define.ContainerDevices, error) {
	// Check for a CDI device
	cdiDevs, err := DeviceFromCDI(device)
	if err == nil {
		return cdiDevs, nil
	}
	
	var devs define.ContainerDevices
	src, dst, permissions, err := Device(device)
	if err != nil {
		return nil, err
	}
	srcInfo, err := os.Stat(src)
	if err != nil {
		return nil, fmt.Errorf("getting info of source device %s: %w", src, err)
	}

	if !srcInfo.IsDir() {
		dev, err := devices.DeviceFromPath(src, permissions)
		if err != nil {
			return nil, fmt.Errorf("%s is not a valid device: %w", src, err)
		}
		dev.Path = dst
		device := define.BuildahDevice{Device: *dev, Source: src, Destination: dst}
		devs = append(devs, device)
		return devs, nil
	}

	// If source device is a directory
	srcDevices, err := devices.GetDevices(src)
	if err != nil {
		return nil, fmt.Errorf("getting source devices from directory %s: %w", src, err)
	}
	for _, d := range srcDevices {
		d.Path = filepath.Join(dst, filepath.Base(d.Path))
		d.Permissions = devices.Permissions(permissions)
		device := define.BuildahDevice{Device: *d, Source: src, Destination: dst}
		devs = append(devs, device)
	}
	return devs, nil
}
