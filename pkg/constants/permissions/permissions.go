// pkg/constants/permissions/permissions.go
package permissions

// Permission represents a single permission string (format: "resource:action")
type Permission string

func (p Permission) String() string {
	return string(p)
}

// Standard Actions (common across resources)
const (
	ActionCreate = "create"
	ActionRead   = "read"
	ActionUpdate = "update"
	ActionDelete = "delete"
	ActionManage = "manage"
)

// Resources
const (
	ResourceAdminUser      = "admin_user"
	ResourceAdminRole      = "admin_role"
	ResourceBanner         = "banner"
	ResourceBlog           = "blog"
	ResourceBrand          = "brand"
	ResourceCategory       = "category"
	ResourceBlogCategory   = "blog_category"
	ResourceConfig         = "config"
	ResourceFooterMenu     = "footer_menu"
	ResourceMenu           = "menu"
	ResourceLog            = "log"
	ResourceMedia          = "media"
	ResourcePCBuild        = "pc_build"
	ResourceProduct        = "product"
	ResourceSpecDefinition = "spec_definition"
	// ResourceSpecDefinitionValue = "spec_definition_value"
	ResourceStatistic = "statistic"
	ResourceUser      = "user"
)

// ============================================================================
// ADMIN USER PERMISSIONS
// ============================================================================
const (
	AdminUserRead           Permission = "admin_user:read"
	AdminUserUpdate         Permission = "admin_user:update"
	AdminUserChangePassword Permission = "admin_user:change_password"
)

// ============================================================================
// ADMIN ROLE PERMISSIONS
// ============================================================================
const (
	AdminRoleCreate Permission = "admin_role:create"
	AdminRoleRead   Permission = "admin_role:read"
	AdminRoleUpdate Permission = "admin_role:update"
	AdminRoleDelete Permission = "admin_role:delete"
	// AdminRoleManage Permission = "admin_role:manage"
)

// ============================================================================
// BANNER PERMISSIONS
// ============================================================================
const (
	BannerCreate Permission = "banner:create"
	BannerRead   Permission = "banner:read"
	BannerUpdate Permission = "banner:update"
	BannerDelete Permission = "banner:delete"
	// BannerManage Permission = "banner:manage"
)

// ============================================================================
// BLOG PERMISSIONS
// ============================================================================
const (
	BlogCreate Permission = "blog:create"
	BlogRead   Permission = "blog:read"
	BlogUpdate Permission = "blog:update"
	BlogDelete Permission = "blog:delete"
)

// ============================================================================
// BLOG CATEGORY PERMISSIONS
// ============================================================================
const (
	BlogCategoryCreate Permission = "blog_category:create"
	BlogCategoryRead   Permission = "blog_category:read"
	BlogCategoryUpdate Permission = "blog_category:update"
	BlogCategoryDelete Permission = "blog_category:delete"
)

// ============================================================================
// BRAND PERMISSIONS
// ============================================================================
const (
	BrandCreate Permission = "brand:create"
	BrandRead   Permission = "brand:read"
	BrandUpdate Permission = "brand:update"
	BrandDelete Permission = "brand:delete"
)

// ============================================================================
// CATEGORY PERMISSIONS
// ============================================================================
const (
	CategoryCreate Permission = "category:create"
	CategoryRead   Permission = "category:read"
	CategoryUpdate Permission = "category:update"
	CategoryDelete Permission = "category:delete"
	// CategoryManage Permission = "category:manage"
)

// ============================================================================
// CONFIG PERMISSIONS (No create/delete - only read/update)
// ============================================================================
const (
// ConfigRead   Permission = "config:read"
// ConfigUpdate Permission = "config:update"
// ConfigManage Permission = "config:manage"
)

// ============================================================================
// FOOTER MENU PERMISSIONS
// ============================================================================
const (
	FooterMenuCreate  Permission = "footer_menu:create"
	FooterMenuRead    Permission = "footer_menu:read"
	FooterMenuUpdate  Permission = "footer_menu:update"
	FooterMenuDelete  Permission = "footer_menu:delete"
	FooterMenuReorder Permission = "footer_menu:reorder"
)

// ============================================================================
// MENU PERMISSIONS
// ============================================================================
const (
	MenuCreate  Permission = "menu:create"
	MenuRead    Permission = "menu:read"
	MenuUpdate  Permission = "menu:update"
	MenuDelete  Permission = "menu:delete"
	MenuReorder Permission = "menu:reorder"
)

// ============================================================================
// MEDIA PERMISSIONS (Custom: upload instead of create)
// ============================================================================
const (
	MediaUpload Permission = "media:upload"
)

// ============================================================================
// LOG PERMISSIONS (Read only - audit logs)
// ============================================================================
const (
	LogRead Permission = "log:read"
)

// ============================================================================
// PC BUILD PERMISSIONS (Admin only has read access)
// ============================================================================
const (
	PCBuildRead Permission = "pc_build:read"
	// PCBuildManage Permission = "pc_build:manage"
)

// ============================================================================
// PRODUCT PERMISSIONS (Custom: export, import, validate_import)
// ============================================================================
const (
	ProductCreate         Permission = "product:create"
	ProductRead           Permission = "product:read"
	ProductUpdate         Permission = "product:update"
	ProductDelete         Permission = "product:delete"
	ProductExport         Permission = "product:export"          // Export products to Excel
	ProductExportAll      Permission = "product:export_all"      // Export all products
	ProductImport         Permission = "product:import"          // Import from Excel
	ProductValidateImport Permission = "product:validate_import" // Validate import file
	// ProductManage         Permission = "product:manage"
)

// ============================================================================
// SPEC DEFINITION PERMISSIONS
// ============================================================================
const (
	SpecDefinitionCreate Permission = "spec_definition:create"
	SpecDefinitionRead   Permission = "spec_definition:read"
	SpecDefinitionUpdate Permission = "spec_definition:update"
	SpecDefinitionDelete Permission = "spec_definition:delete"
	// SpecDefinitionManage Permission = "spec_definition:manage"
)

// ============================================================================
// SPEC DEFINITION VALUE PERMISSIONS
// ============================================================================
const (
// SpecDefinitionValueCreate Permission = "spec_definition_value:create"
// SpecDefinitionValueRead   Permission = "spec_definition_value:read"
// SpecDefinitionValueUpdate Permission = "spec_definition_value:update"
// SpecDefinitionValueDelete Permission = "spec_definition_value:delete"
// SpecDefinitionValueManage Permission = "spec_definition_value:manage"
)

// ============================================================================
// STATISTIC PERMISSIONS (Read only)
// ============================================================================
const (
	StatisticRead Permission = "statistic:read"
	// StatisticManage Permission = "statistic:manage"
)

// ============================================================================
// USER PERMISSIONS (No create - users register themselves)
// ============================================================================
const (
	UserRead       Permission = "user:read"
	UserUpdate     Permission = "user:update"
	UserAssignRole Permission = "user:assign_role"
	UserDelete     Permission = "user:delete"
	// UserManage     Permission = "user:manage"
)

// ============================================================================
// PERMISSION REGISTRY
// ============================================================================

// PermissionInfo contains metadata about a permission
type PermissionInfo struct {
	Permission  Permission
	Resource    string
	Action      string
	Description string
}

// AllPermissions maps resources to their available permissions with descriptions
var AllPermissions = map[string][]PermissionInfo{
	ResourceAdminUser: {
		{AdminUserRead, ResourceAdminUser, ActionRead, "View admin users"},
		{AdminUserUpdate, ResourceAdminUser, ActionUpdate, "Update admin users"},
		{AdminUserChangePassword, ResourceAdminUser, "change_password", "Change admin user password"},
	},
	ResourceAdminRole: {
		{AdminRoleCreate, ResourceAdminRole, ActionCreate, "Create admin roles"},
		{AdminRoleRead, ResourceAdminRole, ActionRead, "View admin roles"},
		{AdminRoleUpdate, ResourceAdminRole, ActionUpdate, "Update admin roles"},
		{AdminRoleDelete, ResourceAdminRole, ActionDelete, "Delete admin roles"},
		// {AdminRoleManage, ResourceAdminRole, ActionManage, "Full access to admin roles"},
	},
	ResourceBanner: {
		{BannerCreate, ResourceBanner, ActionCreate, "Create banners"},
		{BannerRead, ResourceBanner, ActionRead, "View banners"},
		{BannerUpdate, ResourceBanner, ActionUpdate, "Update banners"},
		{BannerDelete, ResourceBanner, ActionDelete, "Delete banners"},
		// {BannerManage, ResourceBanner, ActionManage, "Full access to banners"},
	},
	ResourceBlog: {
		{BlogCreate, ResourceBlog, ActionCreate, "Create blog posts"},
		{BlogRead, ResourceBlog, ActionRead, "View blog posts"},
		{BlogUpdate, ResourceBlog, ActionUpdate, "Update blog posts"},
		{BlogDelete, ResourceBlog, ActionDelete, "Delete blog posts"},
		// {BlogManage, ResourceBlog, ActionManage, "Full access to blog posts"},
	},
	ResourceBrand: {
		{BrandCreate, ResourceBrand, ActionCreate, "Create brands"},
		{BrandRead, ResourceBrand, ActionRead, "View brands"},
		{BrandUpdate, ResourceBrand, ActionUpdate, "Update brands"},
		{BrandDelete, ResourceBrand, ActionDelete, "Delete brands"},
		// {BrandManage, ResourceBrand, ActionManage, "Full access to brands"},
	},
	ResourceCategory: {
		{CategoryCreate, ResourceCategory, ActionCreate, "Create categories"},
		{CategoryRead, ResourceCategory, ActionRead, "View categories"},
		{CategoryUpdate, ResourceCategory, ActionUpdate, "Update categories"},
		{CategoryDelete, ResourceCategory, ActionDelete, "Delete categories"},
		// {CategoryManage, ResourceCategory, ActionManage, "Full access to categories"},
	},
	ResourceBlogCategory: {
		{BlogCategoryCreate, ResourceBlogCategory, ActionCreate, "Create blog categories"},
		{BlogCategoryRead, ResourceBlogCategory, ActionRead, "View blog categories"},
		{BlogCategoryUpdate, ResourceBlogCategory, ActionUpdate, "Update blog categories"},
		{BlogCategoryDelete, ResourceBlogCategory, ActionDelete, "Delete blog categories"},
	},
	// ResourceConfig: {
	// {ConfigRead, ResourceConfig, ActionRead, "View configurations"},
	// {ConfigUpdate, ResourceConfig, ActionUpdate, "Update configurations"},
	// {ConfigManage, ResourceConfig, ActionManage, "Full access to configurations"},
	// },
	ResourceFooterMenu: {
		{FooterMenuCreate, ResourceFooterMenu, ActionCreate, "Create footer menu items"},
		{FooterMenuRead, ResourceFooterMenu, ActionRead, "View footer menu items"},
		{FooterMenuUpdate, ResourceFooterMenu, ActionUpdate, "Update footer menu items"},
		{FooterMenuDelete, ResourceFooterMenu, ActionDelete, "Delete footer menu items"},
		{FooterMenuReorder, ResourceFooterMenu, "reorder", "Reorder footer menu items"},
	},
	ResourceMenu: {
		{MenuCreate, ResourceMenu, ActionCreate, "Create menu items"},
		{MenuRead, ResourceMenu, ActionRead, "View menu items"},
		{MenuUpdate, ResourceMenu, ActionUpdate, "Update menu items"},
		{MenuDelete, ResourceMenu, ActionDelete, "Delete menu items"},
		{MenuReorder, ResourceMenu, "reorder", "Reorder menu items"},
	},
	ResourceMedia: {
		{MediaUpload, ResourceMedia, "upload", "Upload media files"},
	},
	ResourceLog: {
		{LogRead, ResourceLog, ActionRead, "View audit logs"},
	},
	ResourcePCBuild: {
		{PCBuildRead, ResourcePCBuild, ActionRead, "View PC builds"},
		// {PCBuildManage, ResourcePCBuild, ActionManage, "Full access to PC builds"},
	},
	ResourceProduct: {
		{ProductCreate, ResourceProduct, ActionCreate, "Create products"},
		{ProductRead, ResourceProduct, ActionRead, "View products"},
		{ProductUpdate, ResourceProduct, ActionUpdate, "Update products"},
		{ProductDelete, ResourceProduct, ActionDelete, "Delete products"},
		{ProductExport, ResourceProduct, "export", "Export products to Excel"},
		// {ProductExportAll, ResourceProduct, "export_all", "Export all products to Excel"},
		{ProductImport, ResourceProduct, "import", "Import products from Excel"},
		// {ProductValidateImport, ResourceProduct, "validate_import", "Validate product import file"},
		// {ProductManage, ResourceProduct, ActionManage, "Full access to products"},
	},
	ResourceSpecDefinition: {
		{SpecDefinitionCreate, ResourceSpecDefinition, ActionCreate, "Create spec definitions"},
		{SpecDefinitionRead, ResourceSpecDefinition, ActionRead, "View spec definitions"},
		{SpecDefinitionUpdate, ResourceSpecDefinition, ActionUpdate, "Update spec definitions"},
		{SpecDefinitionDelete, ResourceSpecDefinition, ActionDelete, "Delete spec definitions"},
		// {SpecDefinitionManage, ResourceSpecDefinition, ActionManage, "Full access to spec definitions"},
	},
	// ResourceSpecDefinitionValue: {
	// {SpecDefinitionValueCreate, ResourceSpecDefinitionValue, ActionCreate, "Create spec definition values"},
	// {SpecDefinitionValueRead, ResourceSpecDefinitionValue, ActionRead, "View spec definition values"},
	// {SpecDefinitionValueUpdate, ResourceSpecDefinitionValue, ActionUpdate, "Update spec definition values"},
	// {SpecDefinitionValueDelete, ResourceSpecDefinitionValue, ActionDelete, "Delete spec definition values"},
	// {SpecDefinitionValueManage, ResourceSpecDefinitionValue, ActionManage, "Full access to spec definition values"},
	// },
	ResourceStatistic: {
		{StatisticRead, ResourceStatistic, ActionRead, "View statistics"},
		// {StatisticManage, ResourceStatistic, ActionManage, "Full access to statistics"},
	},
	ResourceUser: {
		{UserRead, ResourceUser, ActionRead, "View users"},
		{UserUpdate, ResourceUser, ActionUpdate, "Update users"},
		{UserAssignRole, ResourceUser, "assign_role", "Assign roles to user"},
		{UserDelete, ResourceUser, ActionDelete, "Delete users"},
		// {UserManage, ResourceUser, ActionManage, "Full access to users"},
	},
}

// permissionSet contains all valid permissions for quick lookup
var permissionSet map[string]bool

// resourceActions maps resource to its valid actions (for "manage" permission check)
var resourceActions map[string][]string

func init() {
	permissionSet = make(map[string]bool)
	resourceActions = make(map[string][]string)

	for resource, perms := range AllPermissions {
		actions := make([]string, 0)
		for _, p := range perms {
			permissionSet[p.Permission.String()] = true
			if p.Action != ActionManage {
				actions = append(actions, p.Action)
			}
		}
		resourceActions[resource] = actions
	}
}

// IsValidPermission checks if a permission string is valid
func IsValidPermission(perm string) bool {
	return permissionSet[perm]
}

// GetResourceActions returns all valid actions for a resource (excluding "manage")
func GetResourceActions(resource string) []string {
	return resourceActions[resource]
}

// GetAllPermissionsList returns a flat list of all permission strings
func GetAllPermissionsList() []string {
	result := make([]string, 0, len(permissionSet))
	for perm := range permissionSet {
		result = append(result, perm)
	}
	return result
}

// GetPermissionsForFrontend returns permissions formatted for frontend use
func GetPermissionsForFrontend() map[string][]map[string]string {
	result := make(map[string][]map[string]string)
	for resource, perms := range AllPermissions {
		permList := make([]map[string]string, len(perms))
		for i, p := range perms {
			permList[i] = map[string]string{
				"permission":  p.Permission.String(),
				"action":      p.Action,
				"description": p.Description,
			}
		}
		result[resource] = permList
	}
	return result
}
