package rtmp

import (
	"github.com/gookit/color"
)

var (
	enableBanner = true
	rtmpBanner   = rtmpBanner1

	rtmpBanner1 = `
 ██▀███  ▄▄▄█████▓ ███▄ ▄███▓ ██▓███  
▓██ ▒ ██▒▓  ██▒ ▓▒▓██▒▀█▀ ██▒▓██░  ██▒
▓██ ░▄█ ▒▒ ▓██░ ▒░▓██    ▓██░▓██░ ██▓▒
▒██▀▀█▄  ░ ▓██▓ ░ ▒██    ▒██ ▒██▄█▓▒ ▒
░██▓ ▒██▒  ▒██▒ ░ ▒██▒   ░██▒▒██▒ ░  ░
░ ▒▓ ░▒▓░  ▒ ░░   ░ ▒░   ░  ░▒▓▒░ ░  ░
  ░▒ ░ ▒░    ░    ░  ░      ░░▒ ░     
  ░░   ░   ░      ░      ░   ░░       
   ░                     ░            
=========== version 1.0.0 ============`

	rtmpBanner2 = `
▄▄▄  ▄▄▄▄▄• ▌ ▄ ·.  ▄▄▄·
▀▄ █·•██  ·██ ▐███▪▐█ ▄█
▐▀▀▄  ▐█.▪▐█ ▌▐▌▐█· ██▀·
▐█•█▌ ▐█▌·██ ██▌▐█▌▐█▪·•
.▀  ▀ ▀▀▀ ▀▀  █▪▀▀▀.▀   
===== version 1.0.0 =====`

	rtmpBanner3 = `
           _               _ __  
    _ _   | |_    _ __    | '_ \ 
   | '_|  |  _|  | '  \   | .__/ 
  _|_|_   _\__|  |_|_|_|  |_|__  
_|"""""|_|"""""|_|"""""|_|"""""|_
"'-O-O-'"'-O-O-'"'-O-O-'"'-O-O-'"
========= version 1.0.0 =========`
)

func SetBanner(banner int) {
	if banner == 2 {
		rtmpBanner = rtmpBanner2
	} else if banner == 3 {
		rtmpBanner = rtmpBanner3
	}
}

func SetBannerString(banner string) {
	rtmpBanner = banner
}

func EnableBanner() {
	enableBanner = true
}

func DisableBanner() {
	enableBanner = false
}

func printBanner(addr string) {
	if enableBanner {
		color.Greenln(rtmpBanner)
	}
	color.Greenln("rtmp server running on-> locast" + addr)
}
