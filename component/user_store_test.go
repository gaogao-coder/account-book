/*
Copyright (year) Bytedance Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package component

import "testing"

// TestDefaultUsernameUnique 校验默认用户名随用户 ID 保持唯一。
func TestDefaultUsernameUnique(t *testing.T) {
	first := defaultUsername(1)
	second := defaultUsername(2)

	if first == second {
		t.Fatalf("expected unique usernames, got %q", first)
	}
	if first == "" || second == "" {
		t.Fatalf("expected non-empty usernames")
	}
}
