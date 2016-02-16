
## LE Command Requirements

The following lists the commands and events that a Controller supporting LE shall implement if HCI is claimed.
[Vol 2 Part A.3.19]

#### Mandatory

  - [ ] Command Complete Event (0x0E) [7.7.14]
  - [ ] Command Status Event (0x0F) [7.7.15]
  - [ ] LE Add Device To White List Command (0x08|0x0011) [7.8.16]
  - [ ] LE Clear White List Command (0x08|0x0010) [7.8.15]
  - [ ] LE Read Buffer Size Command (0x08|0x0002) [7.8.2]
  - [ ] Read Local Supported Features Command (0x04|0x0003) [7.4.3]
  - [ ] LE Read Supported States Command (0x08|0x001C) [7.8.27]
  - [ ] LE Read White List Size Command (0x08|0x000F) [7.8.14]
  - [ ] LE Remove Device From White List Command (0x08|0x0012) [7.8.17]
  - [ ] LE Set Event Mask Command (0x08|0x0001) [7.8.1]
  - [ ] LE Test End Command (0x08|0x001F) [7.8.30]
  - [ ] Read BD_ADDR Command (0x04|0x0009) [7.4.6]
  - [ ] LE Read Local Supported Features Command (0x08|0x0003) [7.8.3]
  - [ ] Read Local Version Information Command (0x04|0x0001) [7.4.1]
  - [ ] Reset Command (0x03|0x003) [7.3.2]
  - [ ] Read Local Supported Commands Command (0x04|0x0002) [7.4.2]
  - [ ] Set Event Mask Command (0x03|0x0001) [7.3.1]


#### C1: Mandatory if Controller supports transmitting packets, otherwise optional.

  - [ ] LE Read Advertising Channel Tx Power Command (0x08|0x0007) [7.8.6]
  - [ ] LE Transmitter Test Command (0x08|0x001E) [7.8.29]
  - [ ] LE Set Advertise Enable Command (0x08|0x000A) [7.8.9]
  - [ ] LE Set Advertising Data Command (0x08|0x0008) [7.8.7]
  - [ ] LE Set Advertising Parameters Command (0x08|0x0006) [7.8.5]
  - [ ] LE Set Random Address Command (0x08|0x0005) [7.8.4]


#### C2: Mandatory if Controller supports receiving packets, otherwise optional.

  - [ ] LE Advertising Report Event (0x3E) [7.7.65.2]
  - [ ] LE Receiver Test Command (0x08|0x001D) [7.8.28]
  - [ ] LE Set Scan Enable Command (0x08|0x000C) [7.8.11]
  - [ ] LE Set Scan Parameters Command (0x08|0x000B) [7.8.10]


#### C3: Mandatory if Controller supports transmitting and receiving packets, otherwise optional.

  - [ ] Disconnect Command (0x01|0x0006) [7.1.6]
  - [ ] Disconnection Complete Event (0x05) [7.7.5]
  - [ ] LE Connection Complete Event (0x3E) [7.7.65]
  - [ ] LE Connection Update Command (0x08|0x0013) [7.8.18]
  - [ ] LE Connection Update Complete Event (0x0E) [7.7.65.3]
  - [ ] LE Create Connection Command (0x08|0x000D) [7.8.12]
  - [ ] LE Create Connection Cancel Command (0x08|0x000E) [7.8.13]
  - [ ] LE Read Channel Map Command (0x08|0x0015) [7.8.20]
  - [ ] LE Read Remote Used Features Command (0x08|0x0016) [7.8.21]
  - [ ] LE Read Remote Used Features Complete Event (0x3E) [7.7.65.4]
  - [ ] LE Set Host Channel Classification Command (0x08|0x0014) [7.8.19]
  - [ ] LE Set Scan Response Data Command (0x08|0x0009) [7.8.8]
  - [ ] Host Number Of Completed Packets Command (0x03|0x0035) [7.3.40]
  - [ ] Read Transmit Power Level Command (0x03|0x002D) [7.3.35]
  - [ ] Read Remote Version Information Command (0x01|0x001D) [7.1.23]
  - [ ] Read Remote Version Information Complete Event (0x0C) [7.7.12]
  - [ ] Read RSSI Command (0x05|0x0005) [7.5.4]


#### C4: Mandatory if LE Feature (LL Encryption) is supported otherwise optional.

  - [ ] Encryption Change Event (0x08) [7.7.8]
  - [ ] Encryption Key Refresh Complete Event (0x30) [7.7.39]
  - [ ] LE Encrypt Command (0x08|0x0017) [7.8.22]
  - [ ] LE Long Term Key Request Event (0x3E) [7.7.65.15]
  - [ ] LE Long Term Key Request Reply Command (0x08|0x001A) [7.8.25]
  - [ ] LE Long Term Key Request Negative Reply Command (0x08|0x001B) [7.8.26]
  - [ ] LE Rand Command (0x08|0x0018) [7.8.23]
  - [ ] LE Start Encryption Command (0x08|0x0019) [7.8.24]


#### C5: Mandatory if BR/EDR is supported otherwise optional. [Won't supported]

  - [ ] Read Buffer Size Command [7.4.5]
  - [ ] Read LE Host Support [7.3.78]
  - [ ] Write LE Host Support Command (0x03|0x006D) [7.3.79]


#### C6: Mandatory if LE Feature (Connection Parameters Request procedure) is supported, otherwise optional.

  - [ ] LE Remote Connection Parameter Request Reply Command (0x08|0x0020) [7.8.31]
  - [ ] LE Remote Connection Parameter Request Negative Reply Command (0x08|0x0021) [7.8.32]
  - [ ] LE Remote Connection Parameter Request Event (0x3E) [7.7.65.6]


#### C7: Mandatory if LE Ping is supported otherwise excluded

  - [ ] Write Authenticated Payload Timeout Command (0x01|0x007C) [7.3.94]
  - [ ] Read Authenticated Payload Timeout Command (0x03|0x007B) [7.3.93]
  - [ ] Authenticated Payload Timeout Expired Event (0x57) [7.7.75]
  - [ ] Set Event Mask Page 2 Command (0x03|0x0063) [7.3.69]


#### Optional support

  - [ ] Data Buffer Overflow Event (0x1A) [7.7.26]
  - [ ] Hardware Error Event (0x10) [7.7.16]
  - [ ] Host Buffer Size Command (0x03|0x0033) [7.3.39]
  - [ ] Number Of Completed Packets Event (0x13) [7.7.19]
  - [ ] Set Controller To Host Flow Control Command [7.3.38]


##  Vol 3, Part A, 4 L2CAP Signaling

  - [ ] Command Reject (0x01) [4.1]
  - [ ] Disconnect Request (0x06) [4.6]
  - [ ] Disconnect Response (0x07) [4.7]
  - [ ] Connection Parameter Update Request (0x12) [4.20]
  - [ ] Connection Parameter Update Response (0x13) [4.21]
  - [ ] LE Credit Based Connection Request (0x14) [4.22]
  - [ ] LE Credit Based Connection Response (0x15) [4.23]
  - [ ] LE Flow Control Credit (0x16) [4.24]
