package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

type Instance struct {
	Name    string
	Status  string
	Type    string
	IP      string
	Cluster string
	Info    string
}

const (
	Instances rune = iota + 49
	Clusters
	Details
)

type Cluster struct {
	Name string
}

var dummyInstances = []Instance{}

// var mu = &sync.Mutex{}
var dummyClusters = []Cluster{
	{Name: "cluster-1"},
	{Name: "cluster-2"},
}
var app = tview.NewApplication()
var instanceDetails = getInstanceDetails()
var instances = getInstances()
var clusters = getClusters()

func inputCapture(event *tcell.EventKey) *tcell.EventKey {
	if event.Key() == tcell.KeyEscape {
		app.Stop()
		return nil
	}
	if event.Key() == tcell.KeyLeft {
		app.SetFocus(clusters)
		return nil
	}
	if event.Key() == tcell.KeyRight {
		app.SetFocus(instances)
		return nil
	}
	if event.Key() == tcell.KeyRune && event.Rune() == Instances {
		app.SetFocus(clusters)
		return nil
	}
	if event.Key() == tcell.KeyRune && event.Rune() == Clusters {
		app.SetFocus(instances)
		return nil
	}
	if event.Key() == tcell.KeyRune && event.Rune() == Details {
		app.SetFocus(instanceDetails)
		return nil
	}
	return event
}

func main() {
	instances.SetInputCapture(inputCapture)
	clusters.SetInputCapture(inputCapture)
	instanceDetails.SetInputCapture(inputCapture)
	flex := tview.NewFlex().
		AddItem(clusters, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).AddItem(instances, 0, 2, false).AddItem(instanceDetails, 0, 1, false), 0, 5, false)
	go runInformer()
	if err := app.SetRoot(flex, true).SetFocus(instances).Run(); err != nil {
		panic(err)
	}
}
func getInstances() *tview.Table {
	headers := []string{"Name", "Status", "Type", "IP", "Cluster"}
	table := tview.NewTable().SetBorders(true)
	table.SetSelectable(true, false).SetSelectionChangedFunc(func(row int, column int) {
		// mu.Lock()
		// defer mu.Unlock()
		if row == 0 {
			instanceDetails.SetText("")
			return
		}
		instanceDetails.SetText(fmt.Sprintf("%s\n", dummyInstances[row-1].Info))
	})
	for col, header := range headers {
		table.SetCell(0, col,
			tview.NewTableCell(header).
				SetAlign(tview.AlignCenter).SetExpansion(col))
	}
	for row, instance := range dummyInstances {
		table.SetCell(row+1, 0, tview.NewTableCell(instance.Name).SetExpansion(0).SetAlign(tview.AlignCenter))
		table.SetCell(row+1, 1, tview.NewTableCell(instance.Status).SetExpansion(0).SetAlign(tview.AlignCenter))
		table.SetCell(row+1, 2, tview.NewTableCell(instance.Type).SetExpansion(0).SetAlign(tview.AlignCenter))
		table.SetCell(row+1, 3, tview.NewTableCell(instance.IP).SetExpansion(0).SetAlign(tview.AlignCenter))
		table.SetCell(row+1, 4, tview.NewTableCell(instance.Cluster).SetExpansion(0).SetAlign(tview.AlignCenter))
	}
	table.Select(0, 0).SetFixed(1, 0).SetBorder(true).SetTitle("Instances (2)")
	return table
}

func getClusters() *tview.Table {
	table := tview.NewTable()
	table.SetCell(0, 0, tview.NewTableCell("all").SetExpansion(1).SetAlign(tview.AlignCenter))

	for row, cluster := range dummyClusters {
		table.SetCell(row+1, 0, tview.NewTableCell(cluster.Name).SetExpansion(0).SetAlign(tview.AlignCenter))
	}
	table.SetSelectable(true, false).Select(0, 0).SetFixed(0, 0).SetBorder(true).SetTitle("Clusters (1)").SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
		table.Select(0, 0)
		app.SetFocus(table)
		return action, event
	})
	return table
}

func getInstanceDetails() *tview.TextView {
	text := tview.NewTextView()
	text.SetBorder(true).SetTitle("Details (3)")
	text.SetText("list:\n  - 1\n  - 2").SetScrollable(true).SetWrap(true)
	return text

}

func runInformer() {
	clusterClient, err := getClient()
	if err != nil {
		log.Fatalln(err)
	}

	resource := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(clusterClient, time.Minute, corev1.NamespaceAll, nil)
	informer := factory.ForResource(resource).Informer()

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			u := obj.(*unstructured.Unstructured)
			app.QueueUpdateDraw(func() {
				// mu.Lock()
				// defer mu.Unlock()
				i := Instance{Name: u.GetName(), Status: u.GetKind(), Type: u.GetName(), IP: u.GetAPIVersion(), Cluster: u.GetNamespace()}
				jsonStr, err := yaml.Marshal(u.Object)
				if err != nil {
					log.Print(err)
				}
				i.Info = string(jsonStr)
				dummyInstances = append(dummyInstances, i)
				row := len(dummyInstances)
				instances.SetCell(row, 0, tview.NewTableCell(i.Name).SetExpansion(0).SetAlign(tview.AlignCenter))
				instances.SetCell(row, 1, tview.NewTableCell(i.Status).SetExpansion(0).SetAlign(tview.AlignCenter))
				instances.SetCell(row, 2, tview.NewTableCell(i.Type).SetExpansion(0).SetAlign(tview.AlignCenter))
				instances.SetCell(row, 3, tview.NewTableCell(i.IP).SetExpansion(0).SetAlign(tview.AlignCenter))
				instances.SetCell(row, 4, tview.NewTableCell(i.Cluster).SetExpansion(0).SetAlign(tview.AlignCenter))
			})
		},
		UpdateFunc: func(oldObj, newObj interface{}) {},
		DeleteFunc: func(obj interface{}) {},
	})

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	informer.Run(ctx.Done())
}

func getClient() (*dynamic.DynamicClient, error) {
	kubeConfig := os.Getenv("KUBECONFIG")

	var clusterConfig *rest.Config
	var err error
	if kubeConfig != "" {
		clusterConfig, err = clientcmd.BuildConfigFromFlags("", kubeConfig)
	} else {
		clusterConfig, err = rest.InClusterConfig()
	}
	if err != nil {
		log.Fatalln(err)
	}

	return dynamic.NewForConfig(clusterConfig)
}
