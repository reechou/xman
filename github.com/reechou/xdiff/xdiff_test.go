package xdiff

import (
	"testing"
	"fmt"
)

func TestXDiff(t *testing.T) {
	Xdiff("/Users/reezhou/Desktop/xman/src/youzan/youzan_lb/lb_agent/vips.d", "/Users/reezhou/Desktop/xman/src/youzan/youzan_lb/lb_agent/.backup/2016-02-29_16:19:06/vips.d", "2.html")
	_, r := XdiffToString("/Users/reezhou/Desktop/xman/src/youzan/youzan_lb/lb_agent/keepalived", "/Users/reezhou/Desktop/xman/src/youzan/youzan_lb/lb_agent/.backup/2016-03-29_14:31:50")
	fmt.Println(r)
}
