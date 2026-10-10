package wp

// registerDefaultCacheGroups mirrors wp_start_object_cache group registration (P2.18).
func registerDefaultCacheGroups(cache *ObjectCache) {
	if cache == nil {
		return
	}
	cache.AddGlobalGroups(
		"blog-details",
		"blog-id-cache",
		"blog-lookup",
		"blog_meta",
		"global-posts",
		"image_editor",
		"networks",
		"network-queries",
		"sites",
		"site-details",
		"site-options",
		"site-queries",
		"site-transient",
		"theme_files",
		"translation_files",
		"rss",
		"users",
		"user-queries",
		"user_meta",
		"useremail",
		"userlogins",
		"userslugs",
	)
	cache.AddNonPersistentGroups("counts", "plugins", "theme_json")
}
