import os
import argparse
from pathlib import Path
import json
import datetime
import configparser

def load_settings(config_file):
    """
    Загружает настройки из файла конфигурации.
    Поддерживает форматы JSON и INI.
    
    Args:
        config_file (str): Путь к файлу настроек
    
    Returns:
        dict: Словарь с настройками
    """
    config_path = Path(config_file)
    
    if not config_path.exists():
        raise FileNotFoundError(f"Файл настроек '{config_file}' не найден")
    
    settings = {}
    
    if config_path.suffix.lower() == '.json':
        # Загрузка из JSON
        with open(config_path, 'r', encoding='utf-8') as f:
            settings = json.load(f)
    
    elif config_path.suffix.lower() in ['.ini', '.cfg']:
        # Загрузка из INI
        config = configparser.ConfigParser()
        config.read(config_path, encoding='utf-8')
        
        if 'settings' in config:
            settings_section = config['settings']
            settings = {
                'root_dir': settings_section.get('root_dir', '.'),
                'include_extensions': settings_section.get('include_extensions', '').split(),
                'include_names': settings_section.get('include_names', '').split(),
                'exclude_folders': settings_section.get('exclude_folders', '').split(),
                'exclude_files': settings_section.get('exclude_files', '').split(),
                'case_sensitive': settings_section.getboolean('case_sensitive', False),
                'output_dir': settings_section.get('output_dir', '.')
            }
    
    else:
        raise ValueError(f"Неподдерживаемый формат файла: {config_path.suffix}")
    
    return settings

def find_files(root_dir, include_extensions=None, include_names=None, 
               exclude_folders=None, exclude_files=None, case_sensitive=False):
    """
    Находит файлы в указанной директории и её подпапках с фильтрацией.
    
    Args:
        root_dir (str): Корневая директория для поиска
        include_extensions (list): Список расширений для включения (например, ['.txt', '.py'])
        include_names (list): Список названий файлов для включения
        exclude_folders (list): Список папок для исключения
        exclude_files (list): Список файлов для исключения
        case_sensitive (bool): Чувствительность к регистру
    
    Returns:
        list: Список путей к найденным файлам
    """
    
    # Нормализация параметров
    root_dir = Path(root_dir)
    include_extensions = include_extensions or []
    include_names = include_names or []
    exclude_folders = exclude_folders or []
    exclude_files = exclude_files or []
    
    # Приведение к нижнему регистру если не чувствительно к регистру
    if not case_sensitive:
        include_extensions = [ext.lower() for ext in include_extensions]
        include_names = [name.lower() for name in include_names]
        exclude_folders = [folder.lower() for folder in exclude_folders]
        exclude_files = [file.lower() for file in exclude_files]
    
    found_files = []
    
    try:
        for root, dirs, files in os.walk(root_dir):
            # Исключаем папки из списка исключений
            current_dir = Path(root)
            dir_name = current_dir.name
            
            if not case_sensitive:
                dir_name = dir_name.lower()
            
            # Пропускаем исключенные папки
            folder_matches = any(excluded in str(current_dir) for excluded in exclude_folders)
            if folder_matches:
                continue
            
            # Удаляем исключенные папки из дальнейшего обхода
            dirs[:] = [d for d in dirs if d.lower() not in exclude_folders]
            
            for file in files:
                file_path = current_dir / file
                file_name = file
                file_extension = file_path.suffix
                
                if not case_sensitive:
                    file_name = file_name.lower()
                    file_extension = file_extension.lower()
                
                # Проверяем исключения по имени файла
                if file_name in exclude_files:
                    continue
                
                # Проверяем критерии включения
                should_include = False
                
                # По расширению
                if include_extensions and file_extension in include_extensions:
                    should_include = True
                
                # По имени файла
                if include_names and file_name in include_names:
                    should_include = True
                
                # Если нет критериев включения - включаем все
                if not include_extensions and not include_names:
                    should_include = True
                
                if should_include:
                    found_files.append(str(file_path))
    
    except PermissionError:
        print(f"Ошибка доступа к папке: {root}")
    except Exception as e:
        print(f"Ошибка при обходе папки: {e}")
    
    return found_files

def create_combined_file(found_files, output_dir):
    """
    Создает combined-файл с содержимым найденных файлов.
    
    Args:
        found_files (list): Список путей к файлам
        output_dir (str): Директория для сохранения результата
    
    Returns:
        str: Путь к созданному файлу
    """
    timestamp = datetime.datetime.now().strftime("%Y%m%d-%H%M%S")
    output_path = Path(output_dir) / f"combined-{timestamp}.txt"
    
    try:
        with open(output_path, 'w', encoding='utf-8') as outfile:
            for file_path in found_files:
                # Добавляем разделитель с именем файла
                outfile.write(f"\n{'='*60}\n")
                outfile.write(f"ФАЙЛ: {file_path}\n")
                outfile.write(f"{'='*60}\n\n")
                
                # Читаем и добавляем содержимое файла
                try:
                    with open(file_path, 'r', encoding='utf-8') as infile:
                        content = infile.read()
                        outfile.write(content)
                        # Добавляем перенос строки если файл не заканчивается им
                        if content and not content.endswith('\n'):
                            outfile.write('\n')
                except UnicodeDecodeError:
                    # Пропускаем бинарные файлы
                    outfile.write(f"[БИНАРНЫЙ ФАЙЛ ИЛИ НЕПОДДЕРЖИВАЕМАЯ КОДИРОВКА]\n")
                except Exception as e:
                    outfile.write(f"[ОШИБКА ЧТЕНИЯ: {e}]\n")
        
        print(f"Combined-файл создан: {output_path}")
        return str(output_path)
    
    except Exception as e:
        print(f"Ошибка при создании combined-файла: {e}")
        return None

def main():
    parser = argparse.ArgumentParser(description='Поиск файлов с фильтрацией и создание combined-файла')
    parser.add_argument('--config', help='Файл настроек (JSON или INI форматы)')
    parser.add_argument('--root-dir', help='Корневая директория для поиска')
    parser.add_argument('--ext', nargs='+', help='Расширения файлов для включения (например: .txt .py)')
    parser.add_argument('--names', nargs='+', help='Названия файлов для включения')
    parser.add_argument('--exclude-folders', nargs='+', help='Папки для исключения')
    parser.add_argument('--exclude-files', nargs='+', help='Файлы для исключения')
    parser.add_argument('--case-sensitive', action='store_true', help='Чувствительность к регистру')
    parser.add_argument('--output-dir', help='Директория для сохранения combined-файла')
    parser.add_argument('--list-only', action='store_true', help='Только список файлов, не создавать combined-файл')
    
    args = parser.parse_args()
    
    settings = {}
    
    # Загрузка настроек из файла если указан
    if args.config:
        try:
            settings = load_settings(args.config)
            print(f"Настройки загружены из: {args.config}")
        except Exception as e:
            print(f"Ошибка загрузки настроек: {e}")
            return
    
    # Объединение настроек (аргументы командной строки имеют приоритет)
    root_dir = args.root_dir or settings.get('root_dir', '.')
    include_extensions = args.ext or settings.get('include_extensions', [])
    include_names = args.names or settings.get('include_names', [])
    exclude_folders = args.exclude_folders or settings.get('exclude_folders', [])
    exclude_files = args.exclude_files or settings.get('exclude_files', [])
    case_sensitive = args.case_sensitive or settings.get('case_sensitive', False)
    output_dir = args.output_dir or settings.get('output_dir', '.')
    
    # Проверяем существование корневой директории
    if not os.path.exists(root_dir):
        print(f"Ошибка: Директория '{root_dir}' не существует")
        return
    
    # Создаем директорию для вывода если не существует
    Path(output_dir).mkdir(parents=True, exist_ok=True)
    
    # Поиск файлов
    found_files = find_files(
        root_dir=root_dir,
        include_extensions=include_extensions,
        include_names=include_names,
        exclude_folders=exclude_folders,
        exclude_files=exclude_files,
        case_sensitive=case_sensitive
    )
    
    # Вывод результатов
    if found_files:
        print(f"Найдено файлов: {len(found_files)}")
        for file_path in found_files:
            print(file_path)
        
        # Создание combined-файла если не указано --list-only
        if not args.list_only:
            combined_file = create_combined_file(found_files, output_dir)
            
            # Сохранение списка файлов в отдельный файл
            list_file_path = Path(output_dir) / f"file-list-{datetime.datetime.now().strftime('%Y%m%d-%H%M%S')}.txt"
            with open(list_file_path, 'w', encoding='utf-8') as f:
                for file_path in found_files:
                    f.write(file_path + '\n')
            print(f"Список файлов сохранен в: {list_file_path}")
    else:
        print("Файлы не найдены")

if __name__ == "__main__":
    main()