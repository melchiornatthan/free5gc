import (
	"fmt"
	"net"
	"time"
)

type dhcpKey struct {
	Selection *UPFSelectionParams 
	Supi      string             
}

type dhcpValue struct {
	Addr net.IP `bson:"IP_byte"`
	ExpiresAt time.Time 
}


func SetToDHCPMemoryLocal(cacheKey dhcpKey, cacheValue dhcpValue) {
	DHCPString := cacheKey.Supi + cacheKey.Selection.Dnn + cacheKey.Selection.Dnai + fmt.Sprint(cacheKey.Selection.SNssai.Sst) + cacheKey.Selection.SNssai.Sd
	smfContext.DHCPMemory.Store(DHCPString, cacheValue)
}



func GetFromDHCPMemoryLocal(cacheKey dhcpKey) (dhcpValue, bool) {
	key := cacheKey.Supi + cacheKey.Selection.Dnn + cacheKey.Selection.Dnai + fmt.Sprint(cacheKey.Selection.SNssai.Sst) + cacheKey.Selection.SNssai.Sd

	DHCPValue, ok := smfContext.DHCPMemory.Load(key)
	if !ok {
		return dhcpValue{}, false
	}

	dhcpVal := DHCPValue.(dhcpValue)

	// Check if the DHCP information has expired
	if time.Now().After(dhcpVal.ExpiresAt) {
		// Delete the expired DHCP information
		smfContext.DHCPMemory.Delete(key)
		return dhcpValue{}, false
	}

	// Update the expiration time
	dhcpVal.ExpiresAt = time.Now().Add(24 * time.Hour)
	smfContext.DHCPMemory.Store(key, dhcpVal)

	return dhcpVal, true
}
