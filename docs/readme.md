# Client

### Export

The export service automatically handles the resolution of object relationships between the following objects (if requested):

```text
              ┌────────────────┐   ┌────────────────┐    ┌────────────────┐   ┌────────────────┐
              │                │   │                │    │                │   │                │
              │    Database    ├──▶│      Page      │ ──▶│     Block      │──▶│    Comment     │
              │                │   │                │    │                │   │                │
              └────────────────┘   └────────────────┘    └────────────────┘   └────────────────┘
                      │                    │                     │                     │
                      └────────────────────┴──────────┬──────────┴─────────────────────┘
                                                      │
                                                      ▼
                                              ┌─────────────────┐
                                              │                 │
                                              │      User       │
                                              │                 │
                                              └─────────────────┘
```

| Exportable Object Type       | Description                               |
| ---------------------------- | ----------------------------------------- |
| [Database](#database-export) | A database is a collection of pages.      |
| [Page](#page-export)         | A page is a single page in a database.    |
| [Block](#block-export)       | A block is a single block in a page.      |
| [Comment](#comment-export)   | A comment is a single comment in a block. |
| [User](#user-export)         | A user is a single user.                  |

#### Database Export

Options:

| Option          | Description                                                          |
| --------------- | -------------------------------------------------------------------- |
| `include_pages` | If true, the pages that are linked to the database will be exported. |

#### Page Export

Options:

| Option           | Description                                                          |
| ---------------- | -------------------------------------------------------------------- |
| `include_blocks` | If true, the pages that are linked to the database will be exported. |

#### Block Export

Options:

| Option             | Description                                          |
| ------------------ | ---------------------------------------------------- |
| `include_children` | If true, the children of the block will be exported. |
| `include_comments` | If true, the comments of the block will be exported. |

#### Comment Export

Options:

| Option         | Description                                                           |
| -------------- | --------------------------------------------------------------------- |
| `include_user` | If true, the user that is referenced by the comment will be exported. |

#### User Export

No options are available for the user export.
