package cache

import (
	"net/url"
	"testing"
)

func TestKeyFormats(t *testing.T) {
	cases := []struct {
		got, want string
	}{
		{PublicKey("system", "config"), "serveradmin:v1:system:config"},
		{UserKey("u1", "profile"), "serveradmin:v1:user:u1:profile"},
		{RoleKey("super", "menu"), "serveradmin:v1:role:super:menu"},
		{PublicKey("dict", "country", QueryHash(url.Values{"code": {"CN"}})), "serveradmin:v1:dict:country:" + QueryHash(url.Values{"code": {"CN"}})},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("got %q, want %q", c.got, c.want)
		}
	}
}

func TestQueryHash(t *testing.T) {
	if QueryHash(nil) != "" || QueryHash(url.Values{}) != "" {
		t.Error("空 query 应返回空 hash")
	}
	// 参数顺序不影响结果
	a := QueryHash(url.Values{"page": {"1"}, "size": {"20"}})
	b := QueryHash(url.Values{"size": {"20"}, "page": {"1"}})
	if a != b {
		t.Errorf("参数顺序应不影响 hash: %q != %q", a, b)
	}
	// 空值与 _ 开头的缓存穿透参数应被剔除
	c := QueryHash(url.Values{"page": {"1"}, "kw": {""}, "_t": {"1234567890"}})
	d := QueryHash(url.Values{"page": {"1"}})
	if c != d {
		t.Errorf("空值与 _ 开头参数应被剔除: %q != %q", c, d)
	}
	// 有参与无参必须不同
	if QueryHash(url.Values{"page": {"1"}}) == "" {
		t.Error("有参数时 hash 不应为空")
	}
}

func TestRoleSegment(t *testing.T) {
	if got := RoleSegment([]string{"editor", "admin"}); got != "admin,editor" {
		t.Errorf("多角色应排序拼接: %q", got)
	}
	if got := RoleSegment([]string{"super"}); got != "super" {
		t.Errorf("单角色应直接使用编码: %q", got)
	}
	if got := RoleSegment(nil); got != "-" {
		t.Errorf("空角色应返回占位符: %q", got)
	}
	long := make([]string, 0, 20)
	for i := 0; i < 20; i++ {
		long = append(long, "role-with-a-very-long-name-"+string(rune('a'+i)))
	}
	if got := RoleSegment(long); len(got) != 32 {
		t.Errorf("超长角色列表应返回 32 位十六进制哈希: %q", got)
	}
}
