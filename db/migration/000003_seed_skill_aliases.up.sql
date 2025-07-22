-- =============================================
-- Migration Up: 0003_seed_skill_aliases.up.sql
-- =============================================
-- This migration seeds the database with a large, foundational set of canonical skills
-- and their common aliases. This is crucial for the NLP engine's ability to normalize
-- user input and task descriptions.
-- The script is idempotent; it can be run multiple times without causing errors.

-- Section 1: Seed Canonical Skills
-- -------------------------------------------
-- Insert the official skill names into the `skills` table and mark them as verified.
-- `ON CONFLICT DO NOTHING` prevents errors if these skills already exist.
INSERT INTO skills (skill_name, is_verified)
VALUES
    -- Languages
    ('JavaScript', TRUE), ('Python', TRUE), ('Go', TRUE), ('Java', TRUE),
    ('TypeScript', TRUE), ('C#', TRUE), ('Ruby', TRUE), ('Rust', TRUE), ('PHP', TRUE),
    ('Swift', TRUE), ('Kotlin', TRUE), ('C++', TRUE),
    -- Frontend
    ('React', TRUE), ('Vue.js', TRUE), ('Angular', TRUE), ('Next.js', TRUE), ('Svelte', TRUE),
    ('Tailwind CSS', TRUE), ('Bootstrap', TRUE), ('Sass', TRUE), ('jQuery', TRUE),
    -- Backend
    ('Node.js', TRUE), ('Express.js', TRUE), ('Django REST Framework', TRUE),
    ('Flask', TRUE), ('FastAPI', TRUE), ('Spring Boot', TRUE), ('Ruby on Rails', TRUE), ('.NET', TRUE),
    -- Databases
    ('PostgreSQL', TRUE), ('MongoDB', TRUE), ('Redis', TRUE), ('MySQL', TRUE),
    ('Microsoft SQL Server', TRUE), ('SQLite', TRUE), ('Elasticsearch', TRUE),
    -- DevOps & Cloud
    ('Kubernetes', TRUE), ('Docker', TRUE), ('AWS', TRUE), ('Google Cloud Platform', TRUE),
    ('Microsoft Azure', TRUE), ('CI/CD', TRUE), ('Terraform', TRUE), ('Ansible', TRUE),
    ('Jenkins', TRUE), ('GitHub Actions', TRUE), ('GitLab CI', TRUE), ('Prometheus', TRUE), ('Grafana', TRUE),
    -- Data Science / ML
    ('Scikit-learn', TRUE), ('TensorFlow', TRUE), ('PyTorch', TRUE), ('Keras', TRUE),
    ('Pandas', TRUE), ('NumPy', TRUE), ('Jupyter', TRUE)
ON CONFLICT (skill_name) DO NOTHING;


-- Section 2: Seed Skill Aliases
-- -------------------------------------------
-- Use a Common Table Expression (CTE) to get the IDs of the skills we just inserted.
-- Then, insert all aliases, linking them to the correct skill ID.
WITH skill_ids AS (
    SELECT id, skill_name FROM skills WHERE skill_name IN (
        -- Languages
        'JavaScript', 'Python', 'Go', 'Java', 'TypeScript', 'C#', 'Ruby', 'Rust', 'PHP',
        'Swift', 'Kotlin', 'C++',
        -- Frontend
        'React', 'Vue.js', 'Angular', 'Next.js', 'Svelte', 'Tailwind CSS', 'Bootstrap', 'Sass', 'jQuery',
        -- Backend
        'Node.js', 'Express.js', 'Django REST Framework', 'Flask', 'FastAPI', 'Spring Boot', 'Ruby on Rails', '.NET',
        -- Databases
        'PostgreSQL', 'MongoDB', 'Redis', 'MySQL', 'Microsoft SQL Server', 'SQLite', 'Elasticsearch',
        -- DevOps & Cloud
        'Kubernetes', 'Docker', 'AWS', 'Google Cloud Platform', 'Microsoft Azure', 'CI/CD',
        'Terraform', 'Ansible', 'Jenkins', 'GitHub Actions', 'GitLab CI', 'Prometheus', 'Grafana',
        -- Data Science / ML
        'Scikit-learn', 'TensorFlow', 'PyTorch', 'Keras', 'Pandas', 'NumPy', 'Jupyter'
    )
)
-- Finally, insert all the aliases.
-- `ON CONFLICT DO NOTHING` prevents errors if an alias already exists.
INSERT INTO skill_aliases (alias_name, skill_id)
VALUES
    -- Language Aliases
    ('js', (SELECT id FROM skill_ids WHERE skill_name = 'JavaScript')),
    ('javascripts', (SELECT id FROM skill_ids WHERE skill_name = 'JavaScript')),
    ('es6', (SELECT id FROM skill_ids WHERE skill_name = 'JavaScript')),
    ('esnext', (SELECT id FROM skill_ids WHERE skill_name = 'JavaScript')),
    ('py', (SELECT id FROM skill_ids WHERE skill_name = 'Python')),
    ('python3', (SELECT id FROM skill_ids WHERE skill_name = 'Python')),
    ('golang', (SELECT id FROM skill_ids WHERE skill_name = 'Go')),
    ('java8', (SELECT id FROM skill_ids WHERE skill_name = 'Java')),
    ('java11', (SELECT id FROM skill_ids WHERE skill_name = 'Java')),
    ('ts', (SELECT id FROM skill_ids WHERE skill_name = 'TypeScript')),
    ('csharp', (SELECT id FROM skill_ids WHERE skill_name = 'C#')),
    ('c-sharp', (SELECT id FROM skill_ids WHERE skill_name = 'C#')),
    ('cpp', (SELECT id FROM skill_ids WHERE skill_name = 'C++')),

    -- Frontend Aliases
    ('reactjs', (SELECT id FROM skill_ids WHERE skill_name = 'React')),
    ('react.js', (SELECT id FROM skill_ids WHERE skill_name = 'React')),
    ('vuejs', (SELECT id FROM skill_ids WHERE skill_name = 'Vue.js')),
    ('vue.js', (SELECT id FROM skill_ids WHERE skill_name = 'Vue.js')),
    ('angularjs', (SELECT id FROM skill_ids WHERE skill_name = 'Angular')),
    ('angular js', (SELECT id FROM skill_ids WHERE skill_name = 'Angular')),
    ('angular2+', (SELECT id FROM skill_ids WHERE skill_name = 'Angular')),
    ('next js', (SELECT id FROM skill_ids WHERE skill_name = 'Next.js')),
    ('tailwind', (SELECT id FROM skill_ids WHERE skill_name = 'Tailwind CSS')),
    ('tailwindcss', (SELECT id FROM skill_ids WHERE skill_name = 'Tailwind CSS')),
    ('bootstrap4', (SELECT id FROM skill_ids WHERE skill_name = 'Bootstrap')),
    ('bootstrap5', (SELECT id FROM skill_ids WHERE skill_name = 'Bootstrap')),
    ('scss', (SELECT id FROM skill_ids WHERE skill_name = 'Sass')),

    -- Backend Aliases
    ('node', (SELECT id FROM skill_ids WHERE skill_name = 'Node.js')),
    ('nodejs', (SELECT id FROM skill_ids WHERE skill_name = 'Node.js')),
    ('express', (SELECT id FROM skill_ids WHERE skill_name = 'Express.js')),
    ('express js', (SELECT id FROM skill_ids WHERE skill_name = 'Express.js')),
    ('drf', (SELECT id FROM skill_ids WHERE skill_name = 'Django REST Framework')),
    ('django rest', (SELECT id FROM skill_ids WHERE skill_name = 'Django REST Framework')),
    ('flask api', (SELECT id FROM skill_ids WHERE skill_name = 'Flask')),
    ('spring', (SELECT id FROM skill_ids WHERE skill_name = 'Spring Boot')),
    ('spring framework', (SELECT id FROM skill_ids WHERE skill_name = 'Spring Boot')),
    ('rails', (SELECT id FROM skill_ids WHERE skill_name = 'Ruby on Rails')),
    ('dotnet', (SELECT id FROM skill_ids WHERE skill_name = '.NET')),

    -- Database Aliases
    ('pg', (SELECT id FROM skill_ids WHERE skill_name = 'PostgreSQL')),
    ('postgres', (SELECT id FROM skill_ids WHERE skill_name = 'PostgreSQL')),
    ('postgresql', (SELECT id FROM skill_ids WHERE skill_name = 'PostgreSQL')),
    ('mongo', (SELECT id FROM skill_ids WHERE skill_name = 'MongoDB')),
    ('redis cache', (SELECT id FROM skill_ids WHERE skill_name = 'Redis')),
    ('mysql db', (SELECT id FROM skill_ids WHERE skill_name = 'MySQL')),
    ('ms sql', (SELECT id FROM skill_ids WHERE skill_name = 'Microsoft SQL Server')),
    ('sql server', (SELECT id FROM skill_ids WHERE skill_name = 'Microsoft SQL Server')),

    -- DevOps & Cloud Aliases
    ('k8s', (SELECT id FROM skill_ids WHERE skill_name = 'Kubernetes')),
    ('kube', (SELECT id FROM skill_ids WHERE skill_name = 'Kubernetes')),
    ('docker compose', (SELECT id FROM skill_ids WHERE skill_name = 'Docker')),
    ('amazon web services', (SELECT id FROM skill_ids WHERE skill_name = 'AWS')),
    ('gcp', (SELECT id FROM skill_ids WHERE skill_name = 'Google Cloud Platform')),
    ('google cloud', (SELECT id FROM skill_ids WHERE skill_name = 'Google Cloud Platform')),
    ('azure cloud', (SELECT id FROM skill_ids WHERE skill_name = 'Microsoft Azure')),
    ('cicd', (SELECT id FROM skill_ids WHERE skill_name = 'CI/CD')),
    ('ci-cd', (SELECT id FROM skill_ids WHERE skill_name = 'CI/CD')),
    ('ci/cd', (SELECT id FROM skill_ids WHERE skill_name = 'CI/CD')),
    ('terraform iac', (SELECT id FROM skill_ids WHERE skill_name = 'Terraform')),
    ('ansible config', (SELECT id FROM skill_ids WHERE skill_name = 'Ansible')),
    ('jenkins pipeline', (SELECT id FROM skill_ids WHERE skill_name = 'Jenkins')),

    -- Data Science / ML Aliases
    ('sklearn', (SELECT id FROM skill_ids WHERE skill_name = 'Scikit-learn')),
    ('scikit learn', (SELECT id FROM skill_ids WHERE skill_name = 'Scikit-learn')),
    ('tf', (SELECT id FROM skill_ids WHERE skill_name = 'TensorFlow')),
    ('torch', (SELECT id FROM skill_ids WHERE skill_name = 'PyTorch')),
    ('pandas library', (SELECT id FROM skill_ids WHERE skill_name = 'Pandas')),
    ('numpy', (SELECT id FROM skill_ids WHERE skill_name = 'NumPy')),
    ('jupyter notebooks', (SELECT id FROM skill_ids WHERE skill_name = 'Jupyter'))
ON CONFLICT (alias_name) DO NOTHING;
