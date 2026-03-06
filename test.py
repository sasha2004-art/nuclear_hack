import subprocess
import sys
import os
import io

sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding='utf-8', errors='replace')
sys.stderr = io.TextIOWrapper(sys.stderr.buffer, encoding='utf-8', errors='replace')

def run_go_tests():
    print("[*] Запуск Unit-тестов ядра (Go Core)...")

    base_dir = os.path.dirname(os.path.abspath(__file__))
    core_dir = os.path.join(base_dir, "plotix_core")

    if not os.path.exists(core_dir):
        print(f"[FAIL] Ошибка: Папка ядра не найдена по пути {core_dir}")
        sys.exit(1)

    # FIX: Создаем заглушку для ui_dist, чтобы директива //go:embed не вызывала ошибку компиляции
    ui_dist_dir = os.path.join(core_dir, "ui_dist")
    if not os.path.exists(ui_dist_dir):
        os.makedirs(ui_dist_dir, exist_ok=True)
        with open(os.path.join(ui_dist_dir, "index.html"), "w", encoding="utf-8") as f:
            f.write("<!-- Dummy for tests -->")

    try:
        result = subprocess.run(
            ["go", "test", "-v", "./..."],
            cwd=core_dir,
            capture_output=True,
            text=True,
            encoding='utf-8',
            errors='replace'
        )

        print("-" * 40)
        print(result.stdout)

        if result.stderr:
            print("Предупреждения/Ошибки компиляции:\n", result.stderr)
        print("-" * 40)

        if result.returncode == 0:
            print("[OK] Все Go-тесты пройдены успешно!")
            return True
        else:
            print("[FAIL] Go-тесты провалились. Проверьте логи выше.")
            return False

    except FileNotFoundError:
        print("[FAIL] Ошибка: Не найден компилятор 'go'."
              " Убедитесь, что Golang установлен и добавлен в PATH.")
        sys.exit(1)

if __name__ == "__main__":
    print("====================================")
    print("   Plotix Local - Test Runner")
    print("====================================")

    core_passed = run_go_tests()

    if not core_passed:
        sys.exit(1)
