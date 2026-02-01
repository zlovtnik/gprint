package main

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zlovtnik/gprint/cmd/ui/api"
)

// fetchTimeout is the maximum time to wait for API fetch operations
const fetchTimeout = 10 * time.Second

// fetchAllData returns a batch command that fetches all entity data in parallel
func (m Model) fetchAllData() tea.Cmd {
	return tea.Batch(
		m.fetchCustomers(),
		m.fetchServices(),
		m.fetchContracts(),
		m.fetchPrintJobs(),
	)
}

// API fetch commands with timeout support
func (m Model) fetchCustomers() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		res, err := client.ListCustomersWithContext(ctx, nil)
		if err != nil {
			return errMsg{err}
		}
		return fetchCustomersMsg{res.Items}
	}
}

func (m Model) fetchServices() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		res, err := client.ListServicesWithContext(ctx, nil)
		if err != nil {
			return errMsg{err}
		}
		return fetchServicesMsg{res.Items}
	}
}

func (m Model) fetchContracts() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		res, err := client.ListContractsWithContext(ctx, nil)
		if err != nil {
			return errMsg{err}
		}
		return fetchContractsMsg{res.Items}
	}
}

func (m Model) fetchPrintJobs() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		res, err := client.ListPrintJobsWithContext(ctx, nil)
		if err != nil {
			return errMsg{err}
		}
		return fetchPrintJobsMsg{res.Items}
	}
}

// Customer CRUD commands with timeout context
func (m Model) createCustomer(req *api.CreateCustomerRequest) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		_, err := client.CreateCustomerWithContext(ctx, req)
		if err != nil {
			return errMsg{err}
		}
		return successMsg{"Customer created successfully"}
	}
}

func (m Model) updateCustomer(id int64, req *api.UpdateCustomerRequest) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		_, err := client.UpdateCustomerWithContext(ctx, id, req)
		if err != nil {
			return errMsg{err}
		}
		return successMsg{"Customer updated successfully"}
	}
}

func (m Model) deleteCustomer(id int64) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		if err := client.DeleteCustomerWithContext(ctx, id); err != nil {
			return errMsg{err}
		}
		return successMsg{"Customer deleted successfully"}
	}
}

// Service CRUD commands with timeout context
func (m Model) createService(req *api.CreateServiceRequest) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		_, err := client.CreateServiceWithContext(ctx, req)
		if err != nil {
			return errMsg{err}
		}
		return successMsg{"Service created successfully"}
	}
}

func (m Model) updateService(id int64, req *api.UpdateServiceRequest) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		_, err := client.UpdateServiceWithContext(ctx, id, req)
		if err != nil {
			return errMsg{err}
		}
		return successMsg{"Service updated successfully"}
	}
}

func (m Model) deleteService(id int64) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		if err := client.DeleteServiceWithContext(ctx, id); err != nil {
			return errMsg{err}
		}
		return successMsg{"Service deleted successfully"}
	}
}

// Contract CRUD commands with timeout context
func (m Model) createContract(req *api.CreateContractRequest) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		_, err := client.CreateContractWithContext(ctx, req)
		if err != nil {
			return errMsg{err}
		}
		return successMsg{"Contract created successfully"}
	}
}

func (m Model) updateContract(id int64, req *api.UpdateContractRequest) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		_, err := client.UpdateContractWithContext(ctx, id, req)
		if err != nil {
			return errMsg{err}
		}
		return successMsg{"Contract updated successfully"}
	}
}

func (m Model) generateContract(id int64) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		if err := client.GenerateContractWithContext(ctx, id); err != nil {
			return errMsg{err}
		}
		return successMsg{"Contract generation started"}
	}
}

// createPrintJob creates a print job with the specified format
func (m Model) createPrintJob(id int64, format string) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		_, err := client.CreatePrintJobWithContext(ctx, id, format)
		if err != nil {
			return errMsg{err}
		}
		return successMsg{"Print job created"}
	}
}

func (m Model) signContract(id int64) tea.Cmd {
	client := m.client
	signer := m.signer
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		if err := client.SignContractWithContext(ctx, id, signer); err != nil {
			return errMsg{err}
		}
		return successMsg{"Contract signed"}
	}
}
