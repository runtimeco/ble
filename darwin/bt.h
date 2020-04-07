#ifndef H_BT_
#define H_BT_

#import <Foundation/Foundation.h>
#import <CoreBluetooth/CoreBluetooth.h>

struct byte_arr {
    const uint8_t *data;
    int length;
};

@interface CMgr : NSObject <CBCentralManagerDelegate, CBPeripheralDelegate>
{
@private
    CBCentralManager *_manager;
}

- (uintptr_t) ID;
- (void) scan:(BOOL)allow_dup;
- (void) stopScan;
- (CBPeripheral *) peripheralWithUUID:(NSUUID *)uuid;
- (int) connect:(NSUUID*)peerUUID;
- (int) cancelConnection:(NSUUID *)peerUUID;
- (int) attMTUForPeriphWithUUID:(NSUUID *)peerUUID;
- (int) discoverServices:(NSUUID *)peerUUID services:(NSArray<CBUUID *> *)svcUUIDs;
- (int) discoverCharacteristics:(NSUUID *)peerUUID service:(CBService *)svc
    characteristics:(NSArray<CBUUID *> *)chrUUIDs;
- (int) discoverDescriptors:(NSUUID *)peerUUID characteristic:(CBCharacteristic *)chr;
- (int) readCharacteristic:(NSUUID *)peerUUID characteristic:(CBCharacteristic *)chr;
- (int) writeCharacteristic:(NSUUID *)peerUUID characteristic:(CBCharacteristic *)chr
    value:(struct byte_arr *)val noResponse:(bool)noRsp;
- (int) readDescriptor:(NSUUID *)peerUUID descriptor:(CBDescriptor *)dsc;
- (int) writeDescriptor:(NSUUID *)peerUUID descriptor:(CBDescriptor *)dsc value:(struct byte_arr *)val;
- (int) subscribe:(NSUUID *)peerUUID characteristic:(CBCharacteristic *)chr;
- (int) unsubscribe:(NSUUID *)peerUUID characteristic:(CBCharacteristic *)chr;
- (int) readRSSI:(NSUUID *)peerUUID;
@end

struct discovered_peripheral {
    int rssi;
    const char *local_name;
    struct byte_arr mfg_data;
    int mfg_data_len;
    int power_level;
    int connectable;
    const char **svc_uuids;
    int num_svc_uuids;
    const char **svc_data_uuids;
    struct byte_arr *svc_data_values;
    int num_svc_data;
    struct byte_arr peer_uuid;
};

struct service {
    uintptr_t id;
    const char *uuid;
};

struct characteristic {
    uintptr_t id;
    const char *uuid;
    uint8_t properties;
};

struct descriptor {
    uintptr_t id;
    const char *uuid;
};

void bt_start();
void bt_stop();
void bt_init();

struct byte_arr nsdata_to_byte_arr(const NSData *nsdata);
NSData *byte_arr_to_nsdata(const struct byte_arr *ba);
int dict_int(NSDictionary *dict, NSString *key);
const char *dict_string(NSDictionary *dict, NSString *key);
const void *dict_data(NSDictionary *dict, NSString *key, int *out_len);
const struct byte_arr dict_bytes(NSDictionary *dict, NSString *key);
NSUUID *uuid_from_str(const char *s);

void cb_scan(void *cm, bool allow_dup);
void cb_stop_scan(void *cm);
int cb_connect(void *cm, const char *peer_uuid);
int cb_cancel_connection(void *cm, const char *peer_uuid);
int cb_att_mtu(void *cm, const char *peer_uuid);
int cb_discover_svcs(void *cm, const char *peer_uuid, const char **svc_uuids, int num_svcs);
int cb_discover_chrs(void *cm, const char *peer_uuid, uintptr_t svc_id, const char **chr_uuids, int num_chrs);
int cb_discover_dscs(void *cm, const char *peer_uuid, uintptr_t chr_id);
int cb_read_chr(void *cm, const char *peer_uuid, uintptr_t chr_id);
int cb_write_chr(void *cm, const char *peer_uuid, uintptr_t chr_id, struct byte_arr *val, bool no_rsp);
int cb_read_dsc(void *cm, const char *peer_uuid, uintptr_t dsc_id);
int cb_write_dsc(void *cm, const char *peer_uuid, uintptr_t dsc_id, struct byte_arr *val);
int cb_subscribe(void *cm, const char *peer_uuid, uintptr_t chr_id);
int cb_unsubscribe(void *cm, const char *peer_uuid, uintptr_t chr_id);
int cb_read_rssi(void *cm, const char *peer_uuid);
uintptr_t cb_cmgr_id(void *cm);
CMgr *cb_alloc_cmgr(void);

void BTStateChanged(int enabled, char *msg);
void BTPeripheralDiscovered(uintptr_t cmgr_id, struct discovered_peripheral *dp);
void BTPeripheralConnected(uintptr_t cmgrID, char *uuidStr, int status);
void BTPeripheralDisconnected(uintptr_t cmgrID, char *uuidStr, int reason);
void BTServicesDiscovered(uintptr_t cmgrID, char *uuidStr, int status, struct service *svcs, int numSvcs);
void BTCharacteristicsDiscovered(uintptr_t cmgrID, char *uuidStr, int status, struct characteristic *chrs, int numChrs);
void BTDescriptorsDiscovered(uintptr_t cmgrID, char *uuidStr, int status, struct descriptor *dscs , int numDscs);
void BTCharacteristicRead(uintptr_t cmgrID, char *uuidStr, int status, char *chrUUID, struct byte_arr *chrVal);
void BTCharacteristicWritten(uintptr_t cmgrID, char *uuidStr, int status, char *chrUUID);
void BTDescriptorRead(uintptr_t cmgrID, char *uuidStr, int status, char *dscUUID, struct byte_arr *dscVal);
void BTDescriptorWritten(uintptr_t cmgrID, char *uuidStr, int status, char *dscUUID);
void BTNotificationStateChanged(uintptr_t cmgrID, char *uuidStr, int status, char *chrUUID, bool enabled);
void BTRSSIRead(uintptr_t cmgrID, char *uuidStr, int status, int rssi);

extern dispatch_queue_t bt_queue;

#endif
