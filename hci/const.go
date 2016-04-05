package hci

// HCI Packet types
const (
	pktTypeCommand uint8 = 0x01
	pktTypeACLData uint8 = 0x02
	pktTypeSCOData uint8 = 0x03
	pktTypeEvent   uint8 = 0x04
	pktTypeVendor  uint8 = 0xFF
)

// Event Types [Vol 6 Part B, 2.3 Advertising PDU, 4.4.2].
const (
	advInd        = 0x00 // Connectable undirected advertising (ADV_IND).
	advDirectInd  = 0x01 // Connectable directed advertising (ADV_DIRECT_IND).
	advScanInd    = 0x02 // Scannable undirected advertising (ADV_SCAN_IND).
	advNonconnInd = 0x03 // Non connectable undirected advertising (ADV_NONCONN_IND).
	scanRsp       = 0x04 // Scan Response (SCAN_RSP).
)
