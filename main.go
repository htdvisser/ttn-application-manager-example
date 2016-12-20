package main

import (
	"log"

	"github.com/TheThingsNetwork/ttn/api"
	"github.com/TheThingsNetwork/ttn/api/discovery"
	"github.com/TheThingsNetwork/ttn/api/handler"
	"github.com/TheThingsNetwork/ttn/api/protocol/lorawan"
	"github.com/TheThingsNetwork/ttn/core/types"
	"github.com/TheThingsNetwork/ttn/utils/random"
	"golang.org/x/net/context"
	"google.golang.org/grpc/metadata"
)

// Configuration constants
var (
	DiscoveryAddress = "discover.thethingsnetwork.org:1900"

	// Replace with your Handler's ID:
	HandlerID = "ttn-handler-eu"

	// Replace with your Application Access Key:
	AppAccessKey = "ttn-account-v2.KNsK1bm9TS5ce6Q-4LyC8kQIcDkXGq-s..."

	// Replace these with your AppID, AppEUI and DevEUI:
	AppID  = "test"
	AppEUI = types.AppEUI([8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08})
	DevEUI = types.DevEUI([8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08})

	AppKey = types.AppKey{} // This will be generated in the init() func
)

func init() {
	copy(AppKey[:], random.Bytes(16)) // Generate a random AppKey
}

func getContext() context.Context {
	return metadata.NewContext(context.Background(), metadata.Pairs(
		"service-name", "example-integration", // Replace with the name of your integration
		"service-version", "v2.0.0", // Replace with the version (v2 should correspond with TTN version)
	))
}

func getContextWithKey() context.Context {
	md, _ := metadata.FromContext(getContext())
	return metadata.NewContext(context.Background(), metadata.Join(md, metadata.Pairs(
		"key", AppAccessKey,
	)))
}

func main() {
	discoveryConn, err := api.Dial(DiscoveryAddress)
	if err != nil {
		log.Printf("Error connecting to Discovery server: %s", err.Error())
		return
	}
	defer discoveryConn.Close()

	discoveryClient := discovery.NewDiscoveryClient(discoveryConn)
	handlerAnnouncement, err := discoveryClient.Get(getContext(), &discovery.GetRequest{
		ServiceName: "handler",
		Id:          HandlerID,
	})
	if err != nil {
		log.Printf("Error finding Handler %s in Discovery server: %s", HandlerID, err.Error())
		return
	}

	handlerConn, err := handlerAnnouncement.Dial()
	if err != nil {
		log.Printf("Error connecting to Handler: %s", err.Error())
		return
	}
	applicationManagerClient := handler.NewApplicationManagerClient(handlerConn)

	// We list the devices in the Application:
	devices, err := applicationManagerClient.GetDevicesForApplication(getContextWithKey(), &handler.ApplicationIdentifier{AppId: AppID})
	if err != nil {
		log.Printf("Error getting devices from Handler: %s", err.Error())
		return
	}

	switch len(devices.Devices) {
	case 0:
		log.Print("Application does not have any devices")
	case 1:
		log.Print("Application has one device")
	default:
		log.Printf("Application has %d devices", len(devices.Devices))
	}

	var testDevice *handler.Device

	for _, device := range devices.Devices {
		if device.DevId == "test" {
			testDevice = device
		}
		log.Printf(device.DevId)
	}

	if testDevice == nil {
		// We create a test device
		testDevice = &handler.Device{
			AppId: AppID,
			DevId: "test",
			Device: &handler.Device_LorawanDevice{LorawanDevice: &lorawan.Device{
				AppId:         AppID,
				DevId:         "test",
				AppEui:        &AppEUI,
				DevEui:        &DevEUI,
				AppKey:        &AppKey,
				Uses32BitFCnt: true,
			}},
		}

		_, err := applicationManagerClient.SetDevice(getContextWithKey(), testDevice)
		if err != nil {
			log.Printf("Error creating test device on Handler: %s", err.Error())
			return
		}
		log.Print("Created device \"test\"")
	}

	// Then we update its AppKey:
	testDevice.GetLorawanDevice().AppKey = &AppKey
	_, err = applicationManagerClient.SetDevice(getContextWithKey(), testDevice)
	if err != nil {
		log.Printf("Error updating test device on Handler: %s", err.Error())
		return
	}
	log.Print("Updated device \"test\"")

	// Then we delete it:
	_, err = applicationManagerClient.DeleteDevice(getContextWithKey(), &handler.DeviceIdentifier{AppId: AppID, DevId: "test"})
	if err != nil {
		log.Printf("Error deleting test device on Handler: %s", err.Error())
		return
	}
	log.Print("Deleted device \"test\"")
}
