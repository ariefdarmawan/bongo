package bongo

import (
	"os"

	"github.com/bongo-contrib/loaders"
	"github.com/bongo-contrib/matters"
	"github.com/bongo-contrib/models"
	"github.com/bongo-contrib/renderers"
)

type defaultApp struct {
	loaders.DefaultLoader
	*matters.Matter
	*renderers.DefaultRenderer
}

func newDefaultApp() *defaultApp {
	app := &defaultApp{}
	app.Matter = matters.NewYAML()
	app.DefaultRenderer = renderers.New()
	return app
}

//App is the main bongo application
type App struct {
	gene models.Generator
}

//New creates a new App which uses default models.Generator implementation
func New() *App {
	return NewApp(newDefaultApp())
}

//NewApp creates a new app, that uses g as the generator
func NewApp(g models.Generator) *App {
	return &App{gene: g}
}

// Run runs the app
func (g *App) Run(root string) error {
	files, err := g.gene.Load(root)
	if err != nil {
		return err
	}
	pages := make(models.PageList, len(files))
	send := make(chan *models.Page)
	errs := make(chan error)
	for _, f := range files {
		go func(file string) {
			f, err := os.Open(file)
			if err != nil {
				errs <- err
				return
			}
			defer f.Close()
			front, body, err := g.gene.Parse(f)
			if err != nil {
				errs <- err
				return
			}
			stat, err := f.Stat()
			if err != nil {
				errs <- err
				return
			}
			send <- &models.Page{Path: file, Body: body, Data: front, ModTime: stat.ModTime()}
		}(f)
	}
	n := 0
	var fish error
END:
	for {
		select {
		case pg := <-send:
			pages[n] = pg
			n++
		case perr := <-errs:
			fish = perr
			break END

		default:
			if len(files) <= n {
				break END
			}
		}

	}
	if fish != nil {
		return fish
	}

	// run before rendering
	err = g.gene.Before(root)
	if err != nil {
		return nil
	}
	err = g.gene.Render(root, pages)
	if err != nil {
		renderers.Rollback(root) // roll back before exiting
		return err
	}

	// run after rendering
	err = g.gene.After(root)
	if err != nil {
		return err
	}
	return nil

}
