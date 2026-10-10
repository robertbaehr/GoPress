// link_manager_enabled is permanently disabled in GoPress (P3.48 / P7.25).
package wp

// LinkManagerEnabled always returns false. The classic Links Manager and the
// link_manager_enabled option are out of scope (never enabled).
func LinkManagerEnabled() bool { return false }

const linkManagerEnabledOption = "link_manager_enabled"
