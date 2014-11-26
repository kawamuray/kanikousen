package kanikousen

import (
	"fmt"
	"strings"
)

func ParseSpecOpts(spec string) (string, map[string]string, error) {
	implConfig := strings.SplitN(spec, ":", 2)
	if len(implConfig) != 2 {
		return "", nil, fmt.Errorf("target spec is missing in spec: '%s'", spec)
	}

	opts := make(map[string]string)
	for _, opt := range strings.Split(implConfig[1], ",") {
		kv := strings.SplitN(opt, "=", 2)
		if len(kv) == 2 {
			opts[kv[0]] = kv[1]
		} else {
			opts[kv[0]] = ""
		}
	}
	return implConfig[0], opts, nil
}
