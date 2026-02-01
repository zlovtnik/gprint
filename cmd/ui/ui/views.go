package ui

// ViewState represents the current view
type ViewState int

const (
	ViewMain ViewState = iota
	ViewCustomers
	ViewCustomerDetail
	ViewCustomerCreate
	ViewCustomerEdit
	ViewServices
	ViewServiceDetail
	ViewServiceCreate
	ViewServiceEdit
	ViewContracts
	ViewContractDetail
	ViewContractCreate
	ViewContractEdit
	ViewPrintJobs
	ViewPrintJobDetail
	ViewSettings
	ViewLogin
)

// MenuItem represents a menu item
type MenuItem struct {
	Title       string
	Description string
	View        ViewState
}

// mainMenuItems is the internal slice of menu items
var mainMenuItems = []MenuItem{
	{Title: "Customers", Description: "Manage customers", View: ViewCustomers},
	{Title: "Services", Description: "Manage service catalog", View: ViewServices},
	{Title: "Contracts", Description: "Manage contracts", View: ViewContracts},
	{Title: "Print Jobs", Description: "View print job queue", View: ViewPrintJobs},
	{Title: "Settings", Description: "Configure application", View: ViewSettings},
}

// GetMainMenuItems returns a copy of the main menu items to prevent mutation
func GetMainMenuItems() []MenuItem {
	items := make([]MenuItem, len(mainMenuItems))
	copy(items, mainMenuItems)
	return items
}

// CRUDAction represents a CRUD action
type CRUDAction int

const (
	ActionList CRUDAction = iota
	ActionView
	ActionCreate
	ActionEdit
	ActionDelete
	ActionBack // Navigation back action
)

// entityMenuItems are the entity CRUD menu options (unexported)
var entityMenuItems = []MenuItem{
	{Title: "List All", Description: "View all records"},
	{Title: "View Details", Description: "View a specific record"},
	{Title: "Create New", Description: "Create a new record"},
	{Title: "Edit", Description: "Edit an existing record"},
	{Title: "Delete", Description: "Delete a record"},
	{Title: "Back", Description: "Return to main menu"},
}

// GetEntityMenuItems returns a copy of the entity menu items to prevent mutation
func GetEntityMenuItems() []MenuItem {
	items := make([]MenuItem, len(entityMenuItems))
	copy(items, entityMenuItems)
	return items
}
