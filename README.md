hems
====

B ルート対応 Wi-SUN デバイスにシリアル通信で接続し、消費電力を取得します。

## Features
- [x] 瞬間消費電力の取得
- [ ] 積算消費電力の取得

## Support Devices
- [x] [UDG-1-WSNE](https://web116.jp/shop/netki/miruene_usb/miruene_usb_00.html)
- [x] [BP35C0](https://www.rohm.co.jp/products/wireless-communication/specified-low-power-radio-modules/bp35c0-product), [BP35C2](https://www.rohm.co.jp/products/wireless-communication/specified-low-power-radio-modules/bp35c2-product)

## Usage

※ 本アプリケーションは、日本の電波法に準拠しているデバイスで、新たに免許等が必要ないものを接続して、利用する事を想定しています。

## Build

### Raspberry Pi 4

```sh
make build-rpi4
```

### Raspberry Pi 2, 3 or Zero

```sh
make build-rpi
```

## Example
`example` ディレクトリ以下を参照してください。
