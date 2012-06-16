Tumblerous Plan
===============

* Bookmarklet (JavaScript)
    * Extract content snippets from page
        * Selected area if user selected one before opening bookmarklet
        * Chosen heuristically if no area selected
        * Posts raw HTML snippet, not Markdown
    * Pre-populate title from page title but allow user edit
    * Post title and content to API to create new post on form submission
* Markdown -> HTML
    * Use Github-flavored Markdown
* User login (Go)
    * OpenID is fine for this
* User logout (Go)
* Create post (Go)
    * Requires title and content
        * Content can be either
            * raw HTML to support submission via bookmarklet
            * Markdown (for submission via Web app)
    * Background
        * Regenerate index page
        * Regenerate archive page
        * Regenerate feed XML
* Delete post (Go)
    * Requires only post ID
    * Background
        * Regenerate index page (if necessary)
        * Regenerate archive page
        * Regenerate feed XML (if necessary)
* Update post (Go)
    * Requires params as 'Create post', plus post ID
    * Background, if title changes
        * Regenerate index page (if necessary)
        * Regenerate archive page
        * Regenerate feed XML (if necessary)

