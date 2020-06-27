package provision

import "testing"

func TestCustomGCEIDConstAndDest(t *testing.T) {
	inputInstanceName := "inlets"
	inputZone := "us-central1-a"
	inputProjectID := "playground"

	customID := toGCEID(inputInstanceName, inputZone, inputProjectID)

	outputInstanceName, outputZone, outputProjectID, err := getGCEFieldsFromID(customID)
	if err != nil {
		t.Error(err)
	}
	if inputInstanceName != outputInstanceName ||
		inputZone != outputZone ||
		inputProjectID != outputProjectID {
		t.Errorf("Input fields: %s, %s, %s are not equal to the ouput fields: %s, %s, %s",
			inputInstanceName, inputZone, inputProjectID, outputInstanceName, outputZone, outputProjectID)
	}

}
