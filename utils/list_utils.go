package utils

func List2map(list []string) map[string]string {
	mp := make(map[string]string)
	for _, item := range list {
		mp[item] = item
	}
	return mp
}
