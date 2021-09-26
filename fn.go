// Copyright 2016 Evans. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cedar

func isReduced(reduced ...bool) bool {
	if len(reduced) > 0 && !reduced[0] {
		return false
	}

	return true
}
