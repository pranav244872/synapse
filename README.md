# 🧠 Synapse – Smart Task & Skill Matching Platform

**One-liner:**

An intelligent project management tool that automatically recommends the best-fit engineers for tasks based on skills, experience, and availability—powered by machine learning and real-time collaboration.

---

### ✅ Core Goals

- Automate task assignment using skill and performance-based recommendations.
- Eliminate manual bottlenecks in project management with ML-driven decision support.

---

### 🧩 MVP Features

#### 🔐 User Onboarding

- Engineers are added via a form + resume upload.
- Gemini API extracts and normalizes skills + proficiency levels.
- Parsed skills are stored in PostgreSQL; unknown ones are added as `unverified`.

#### 📝 Task Creation

- Tasks include title + description.
- Gemini API extracts required skills via NLP.
- Skills are normalized and linked to the task.

#### 🤖 ML-Based Recommendations

- A Python microservice (using `surprise`) generates top engineer suggestions for each task.
- **Hybrid Recommendation Engine**:
    - *Content-Based Filtering*: Matches required task skills to user profiles.
    - *Collaborative Filtering*: Suggests users based on past task performance patterns.
- The Go backend queries this model for real-time recommendations.

🧠 **[Recommender Service GitHub Repo](https://github.com/pranav244872/synapse-recommender)**

#### 🔄 Task Assignment Workflow

- Assigning a task:
    - Sets task status to `in_progress`
    - Marks engineer as `busy`
    - All updates run as a PostgreSQL transaction (`sqlc` + `pgx`)

#### ✅ Task Completion Flow

- Marks task as `done` with a timestamp.
- Restores engineer’s availability to `available`.
- Logs completion for future model retraining.

---

### ⚙️ Tech Stack

| Component         | Tech                                                         |
|------------------|--------------------------------------------------------------|
| Backend           | Go + Gin                                                     |
| Database          | PostgreSQL + `sqlc` + `pgx` (type-safe queries, transactions)|
| ML Recommender    | Python + `surprise` (`SVD`)                                  |
| NLP & Skill Extraction | Gemini API                                              |
| Realtime Updates  | Event-driven architecture (planned)                          |

---

### 🧩 Related Repositories

- 🔮 **Frontend**: [synapse-frontend](https://github.com/pranav244872/synapse-frontend)
- 🧠 **Recommender Service**: [synapse-recommender](https://github.com/pranav244872/synapse-recommender)

---

### 📚 Full Documentation

For implementation details, architecture, and API references, check the [Synapse Notion Wiki](https://tropical-whitefish-023.notion.site/Project-234d3f155a0d80278442d35f7cdb918f?source=copy_link).
