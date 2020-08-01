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

### Installation (Ubuntu)

#### Deploy binary

armv7 用のバイナリを使う場合は次のようになります。

```sh
sudo cp ./hems_linux_armv7 /usr/local/bin/
```

#### Deploy service and udev rule

`hems.service` で設定されている以下の項目は適切な値に変更してください。

```
Environment=HEMS_ROUTEB_ID=xxx
Environment=HEMS_PASSWORD=xxx
Environment=HEMS_DEVICE=/dev/udg-1-wsne
ExecStart=/usr/local/bin/hems_linux_armv7
```

```sh
sudo cp ./example/etc/systemd/system/hems.service /etc/systemd/system/hems.service
sudo systemctl daemon-reload
```

`UDG-1-WSNE` 以外を使う場合は `99-hems.rules` を修正する必要があります。

```sh
sudo cp ./example/etc/udev/rules.d/99-hems.rules /etc/udev/rules.d/99-hems.rules
sudo udevadm control --reload
```

USB ドングルを接続すると自動的に hems service が起動します。
USB ドングルを外すと自動的に停止します。

## Build

### Raspberry Pi 4

```sh
make build-rpi4
```

### Raspberry Pi 2, 3

```sh
make build-rpi2
```

### Raspberry Pi 1, Zero

```sh
make build-rpi0
```

## Example
`example` ディレクトリ以下を参照してください。
