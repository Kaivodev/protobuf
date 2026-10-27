// Command policycheck fails when any RPC in a Buf image lacks a valid
// (aipay.authz.v1.policy). Usage: policycheck <image.binpb>
package main

import (
	"fmt"
	"os"

	"github.com/aipay-dev/protobuf/internal/policycheck"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: policycheck <image.binpb>")
		os.Exit(2)
	}
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "read descriptor image:", err)
		os.Exit(2)
	}
	// A Buf image is wire-compatible with FileDescriptorSet.
	var set descriptorpb.FileDescriptorSet
	if err = proto.Unmarshal(raw, &set); err != nil {
		fmt.Fprintln(os.Stderr, "decode descriptor image:", err)
		os.Exit(2)
	}
	problems := policycheck.Check(&set)
	for _, problem := range problems {
		fmt.Fprintln(os.Stderr, problem)
	}
	if len(problems) != 0 {
		fmt.Fprintf(os.Stderr, "%d method authorization policy problem(s)\n", len(problems))
		os.Exit(1)
	}
}
