-- =============================================
-- Migration Down: 0003_seed_skill_aliases.down.sql
-- =============================================
-- This migration reverses the seeding of skill aliases by deleting only the aliases
-- added in the corresponding 'up' script. The canonical skills in the 'skills' table
-- are intentionally left untouched, as they may be referenced by other data.

-- Delete only the aliases that were added by the `up` migration.
DELETE FROM skill_aliases WHERE alias_name IN (
    -- Language Aliases
    'js', 'javascripts', 'es6', 'esnext', 'py', 'python3', 'golang', 'java8', 'java11', 'ts', 'csharp', 'c-sharp', 'cpp',
    -- Frontend Aliases
    'reactjs', 'react.js', 'vuejs', 'vue.js', 'angularjs', 'angular js', 'angular2+', 'next js', 'tailwind', 'tailwindcss', 'bootstrap4', 'bootstrap5', 'scss',
    -- Backend Aliases
    'node', 'nodejs', 'express', 'express js', 'drf', 'django rest', 'flask api', 'spring', 'spring framework', 'rails', 'dotnet',
    -- Database Aliases
    'pg', 'postgres', 'postgresql', 'mongo', 'redis cache', 'mysql db', 'ms sql', 'sql server',
    -- DevOps & Cloud Aliases
    'k8s', 'kube', 'docker compose', 'amazon web services', 'gcp', 'google cloud', 'azure cloud', 'cicd', 'ci-cd', 'ci/cd', 'terraform iac', 'ansible config', 'jenkins pipeline',
    -- Data Science / ML Aliases
    'sklearn', 'scikit learn', 'tf', 'torch', 'pandas library', 'numpy', 'jupyter notebooks'
);

-- To completely reverse the seeding, including the canonical skills, uncomment the following block.
-- Warning: This is a destructive action and may fail if these skills are referenced by other tables (e.g., user_skills, task_required_skills).
DELETE FROM skills WHERE skill_name IN (
    'JavaScript', 'Python', 'Go', 'Java', 'TypeScript', 'C#', 'Ruby', 'Rust', 'PHP',
    'Swift', 'Kotlin', 'C++', 'React', 'Vue.js', 'Angular', 'Next.js', 'Svelte',
    'Tailwind CSS', 'Bootstrap', 'Sass', 'jQuery', 'Node.js', 'Express.js',
    'Django REST Framework', 'Flask', 'FastAPI', 'Spring Boot', 'Ruby on Rails', '.NET',
    'PostgreSQL', 'MongoDB', 'Redis', 'MySQL', 'Microsoft SQL Server', 'SQLite',
    'Elasticsearch', 'Kubernetes', 'Docker', 'AWS', 'Google Cloud Platform',
    'Microsoft Azure', 'CI/CD', 'Terraform', 'Ansible', 'Jenkins', 'GitHub Actions',
    'GitLab CI', 'Prometheus', 'Grafana', 'Scikit-learn', 'TensorFlow', 'PyTorch',
    'Keras', 'Pandas', 'NumPy', 'Jupyter'
);
