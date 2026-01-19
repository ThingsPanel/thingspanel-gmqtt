package thingspanel

func deviceIDFromClient(clientID string) (string, error) {
	if clientID == "" {
		return "", nil
	}
	return GetStr("mqtt_clinet_id_" + clientID)
}
