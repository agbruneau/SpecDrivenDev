package corpus

import "net"

func loadRoute() { _, _ = net.Dial("tcp", "localhost:0") }
