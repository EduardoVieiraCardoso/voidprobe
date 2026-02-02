# Como Criar Repositório Privado no GitHub

## ✅ Já Feito

- [x] Git inicializado
- [x] Arquivos commitados localmente
- [x] Pronto para push

## 🚀 Próximos Passos

### Opção 1: Via Interface Web (Mais Fácil)

1. **Acesse**: https://github.com/new

2. **Preencha**:
   - Repository name: `voidprobe`
   - Description: `Secure reverse tunnel for authorized remote administration`
   - Visibilidade: ✅ **Private** (importante!)
   - ❌ NÃO marque "Add a README file"
   - ❌ NÃO adicione .gitignore
   - ❌ NÃO escolha license

3. **Clique**: "Create repository"

4. **Na tela seguinte**, copie seu nome de usuário GitHub e execute:

```bash
cd "C:\Users\Eduardo Vieira\Desktop\teste\voidprobe"

# Substituir SEU_USUARIO pelo seu nome no GitHub
git remote add origin https://github.com/SEU_USUARIO/voidprobe.git

# Renomear branch para main
git branch -M main

# Fazer push
git push -u origin main
```

### Opção 2: Via GitHub CLI (Requer Instalação)

Se quiser instalar o GitHub CLI:

```bash
# Instalar
winget install --id GitHub.cli

# Autenticar
gh auth login

# Criar e fazer push
gh repo create voidprobe --private --source=. --remote=origin --push
```

## 🔐 Autenticação

### Para HTTPS (Recomendado)

Se usar `https://github.com/...`, você precisará de um **Personal Access Token**:

1. Acesse: https://github.com/settings/tokens
2. Clique: "Generate new token" → "Generate new token (classic)"
3. Nome: `voidprobe-upload`
4. Scopes: Marque `repo` (todos os sub-items)
5. Clique: "Generate token"
6. **COPIE O TOKEN** (não aparecerá novamente!)

Quando fizer `git push`, use:
- Username: seu nome de usuário GitHub
- Password: o token copiado

### Para SSH (Alternativa)

Se preferir SSH:

1. Gerar chave SSH (se não tiver):
```bash
ssh-keygen -t ed25519 -C "seu-email@example.com"
```

2. Copiar chave pública:
```bash
cat ~/.ssh/id_ed25519.pub
```

3. Adicionar em: https://github.com/settings/keys

4. Usar URL SSH:
```bash
git remote add origin git@github.com:SEU_USUARIO/voidprobe.git
```

## 📋 Comandos Resumidos

```bash
# 1. Criar repo no GitHub (via web)
#    https://github.com/new

# 2. Adicionar remote
cd "C:\Users\Eduardo Vieira\Desktop\teste\voidprobe"
git remote add origin https://github.com/SEU_USUARIO/voidprobe.git

# 3. Renomear branch
git branch -M main

# 4. Push
git push -u origin main
```

## ✅ Verificar Sucesso

Após o push, acesse:
```
https://github.com/SEU_USUARIO/voidprobe
```

Você deve ver:
- ✅ Repositório privado (🔒 Private)
- ✅ Todos os arquivos
- ✅ README.md renderizado
- ✅ Pastas `server/` e `client/`

## 🎉 Pronto!

Seu repositório privado está criado e sincronizado!

---

**Comandos úteis:**

```bash
# Ver status
git status

# Ver remotes
git remote -v

# Fazer novos commits
git add .
git commit -m "sua mensagem"
git push

# Clonar em outra máquina
git clone https://github.com/SEU_USUARIO/voidprobe.git
```
