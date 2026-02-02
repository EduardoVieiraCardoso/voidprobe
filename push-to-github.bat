@echo off
echo ==========================================
echo   Push para GitHub (Repositorio Privado)
echo ==========================================
echo.

REM Mudar para o diretorio do projeto
cd /d "%~dp0"

echo [1/5] Verificando Git...
git --version >nul 2>&1
if errorlevel 1 (
    echo ERRO: Git nao encontrado!
    echo Instale em: https://git-scm.com/download/win
    pause
    exit /b 1
)
echo OK: Git instalado
echo.

echo [2/5] Verificando status...
git status
echo.

echo ==========================================
echo   INSTRUCOES
echo ==========================================
echo.
echo 1. Acesse: https://github.com/new
echo 2. Repository name: voidprobe
echo 3. Marque: Private
echo 4. Clique: Create repository
echo.
echo Depois volte aqui!
echo.
pause

echo.
echo [3/5] Digite seu nome de usuario do GitHub:
set /p GITHUB_USER="Usuario GitHub: "

if "%GITHUB_USER%"=="" (
    echo ERRO: Usuario nao pode ser vazio!
    pause
    exit /b 1
)

echo.
echo [4/5] Configurando remote...
git remote remove origin 2>nul
git remote add origin https://github.com/%GITHUB_USER%/voidprobe.git
git branch -M main

echo.
echo [5/5] Fazendo push...
echo.
echo IMPORTANTE: Quando pedir senha, use um Personal Access Token!
echo Crie em: https://github.com/settings/tokens
echo.
git push -u origin main

if errorlevel 0 (
    echo.
    echo ==========================================
    echo   SUCESSO!
    echo ==========================================
    echo.
    echo Repositorio criado em:
    echo https://github.com/%GITHUB_USER%/voidprobe
    echo.
) else (
    echo.
    echo ==========================================
    echo   ERRO ao fazer push
    echo ==========================================
    echo.
    echo Se pediu autenticacao:
    echo 1. Acesse: https://github.com/settings/tokens
    echo 2. Generate new token (classic)
    echo 3. Marque: repo (todas as opcoes)
    echo 4. Copie o token
    echo 5. Use o token como senha
    echo.
)

pause
